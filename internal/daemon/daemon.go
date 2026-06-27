// Package daemon is the wake layer's runtime: it tails the shared chat bus,
// asks the pure wake policy who to wake, and drives the per-kind Runner to
// resume those teammates. It is delivery, never work-orchestration — every wake
// traces to a message someone sent (see package wake).
//
// The Dispatcher.Tick method is the testable unit; `hatch watch` just calls it
// on an interval. Runners are dispatched on background goroutines so a slow
// teammate does not block delivery to others, and so the "member is mid-turn"
// debounce (wake Rule 3) is real across ticks.
package daemon

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fioenix/hatch/internal/bus"
	"github.com/fioenix/hatch/internal/model"
	"github.com/fioenix/hatch/internal/roster"
	"github.com/fioenix/hatch/internal/wake"
)

// Runner wakes one member with the messages that triggered it. Implementations
// resume the member's CLI session (production) or record the call (tests).
type Runner interface {
	Wake(m model.Member, payload []model.Message) error
}

// Dispatcher turns new bus messages into wakes. It is single-goroutine for Tick
// but tracks in-flight runners (working set) under a mutex so debounce works.
type Dispatcher struct {
	Bus    *bus.Bus
	Roster *roster.Store
	Runner Runner
	Cfg    wake.Config

	mu      sync.Mutex
	wg      sync.WaitGroup             // in-flight dispatch goroutines (for drain)
	working map[string]bool            // members with an in-flight Runner.Wake
	state   wake.State                 // wake policy memory across ticks
	pending map[string][]model.Message // held payloads awaiting redelivery
	cursor  time.Time                  // newest message TS already processed
}

// Wait blocks until all in-flight wakes finish (graceful drain; used on shutdown
// and in tests so async file writes settle).
func (d *Dispatcher) Wait() { d.wg.Wait() }

// New returns a Dispatcher ready to Tick.
func New(b *bus.Bus, rs *roster.Store, r Runner, cfg wake.Config) *Dispatcher {
	d := &Dispatcher{
		Bus: b, Roster: rs, Runner: r, Cfg: cfg,
		working: map[string]bool{},
		state:   freshState(),
		pending: map[string][]model.Message{},
	}
	d.cursor = d.loadCursor() // survive restarts: don't replay the whole backlog
	return d
}

func freshState() wake.State {
	return wake.State{
		Working: map[string]bool{}, Rate: map[string][]time.Time{}, Depth: map[string]int{},
		PingPong: map[string]int{}, LastTrig: map[string]string{}, OpenAsks: map[string]string{}, Progress: map[string]bool{},
	}
}

// Tick processes one delivery cycle: read messages newer than the cursor, run
// the wake policy, dispatch ready wakes (plus any now-deliverable held ones),
// and post escalations to the boss. It returns what it dispatched/escalated so
// `hatch watch` and tests can observe it.
func (d *Dispatcher) Tick(now time.Time) (dispatched []model.WakeDecision, escalations []model.Escalation, err error) {
	r, err := d.Roster.Effective(now)
	if err != nil {
		return nil, nil, err
	}

	newMsgs, maxTS, err := d.tail()
	if err != nil {
		return nil, nil, err
	}
	if maxTS.After(d.cursor) {
		d.cursor = maxTS
		d.saveCursor()
	}

	// Replace (not merge) the working snapshot so a finished runner stops
	// debouncing future wakes; recompute open asks for the policy.
	d.state.Working = d.snapshotWorking()
	d.state.OpenAsks = d.openAsks()

	decisions, esc, next := wake.Decide(r, newMsgs, d.state, d.Cfg, now)
	d.state = next

	// Merge held payloads from earlier ticks, then dispatch or re-hold. Wakes are
	// keyed by (agent, thread): each task thread carries its own session, so they
	// are never merged. Dispatch stays serial per agent (one in-flight runner per
	// member) — concurrent runs in one repo would race the working tree; true
	// per-thread parallelism needs worktree isolation (a separate change).
	handled := map[string]bool{}
	for i := range decisions {
		dec := decisions[i]
		ck := threadKey(dec.Agent, dec.Thread)
		handled[ck] = true
		if buf := d.takePending(ck); len(buf) > 0 {
			dec.Payload = append(append([]model.Message{}, buf...), dec.Payload...)
		}
		if dec.Hold == model.HoldNone && !d.isWorking(dec.Agent) {
			d.dispatch(r[dec.Agent], dec.Thread, dec.Payload)
			dispatched = append(dispatched, dec)
		} else {
			d.holdPending(ck, dec.Payload)
		}
	}

	// Flush threads that became free and still have buffered payloads.
	for _, ck := range d.pendingKeys() {
		if handled[ck] {
			continue // re-held above this tick (e.g. rate/working); don't override
		}
		agent, thread := splitThreadKey(ck)
		if d.isWorking(agent) {
			continue
		}
		if !r.Reachable(agent) {
			continue // member left/offline since it was queued; hold until reachable again
		}
		buf := d.takePending(ck)
		if len(buf) == 0 {
			continue
		}
		d.dispatch(r[agent], thread, buf)
		dispatched = append(dispatched, model.WakeDecision{Agent: agent, Thread: thread, Reason: model.WakeMention, Payload: buf})
	}

	for _, e := range esc {
		if perr := d.postEscalation(e); perr == nil {
			escalations = append(escalations, e)
		}
	}
	return dispatched, escalations, nil
}

// dispatch runs a Runner on a background goroutine, marking the member working
// for the duration and suspended on completion. A member with no session keeps
// running (fresh contact); presence errors are non-fatal.
func (d *Dispatcher) dispatch(m model.Member, thread string, payload []model.Message) {
	if m.ID == "" {
		return
	}
	d.setWorking(m.ID, true)
	_ = d.Roster.SetStatus(m.ID, model.MemberOnline)
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		err := d.Runner.Wake(m, payload)
		// On failure, requeue (keyed by this thread) before clearing the working
		// flag so the next tick redelivers it — a spawn/resume error never
		// silently drops the message.
		if err != nil {
			d.requeue(threadKey(m.ID, thread), payload)
		}
		d.setWorking(m.ID, false)
		_ = d.Roster.SetStatus(m.ID, model.MemberSuspended)
	}()
}

func (d *Dispatcher) postEscalation(e model.Escalation) error {
	_, err := d.Bus.Post(model.Message{
		Channel: "dm-hatch-" + e.To,
		From:    "hatch",
		To:      []string{e.To},
		Type:    model.MsgText,
		Body:    "⚠ escalation (" + e.Cause + ") on episode " + e.Episode + ": " + e.Note,
	})
	return err
}

// tail returns messages newer than the cursor across all threads, sorted by TS.
func (d *Dispatcher) tail() ([]model.Message, time.Time, error) {
	chans, err := d.Bus.Channels()
	if err != nil {
		return nil, time.Time{}, err
	}
	var out []model.Message
	maxTS := d.cursor
	for _, ch := range chans {
		msgs, err := d.Bus.Messages(ch)
		if err != nil {
			return nil, time.Time{}, err
		}
		for _, m := range msgs {
			t, perr := time.Parse(time.RFC3339Nano, m.TS)
			if perr != nil {
				continue
			}
			if t.After(d.cursor) {
				out = append(out, m)
			}
			if t.After(maxTS) {
				maxTS = t
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out, maxTS, nil
}

// openAsks builds the open-question map for the wake policy: an ask is open
// until a reply lands on it. Computed from full history each tick (bounded).
func (d *Dispatcher) openAsks() map[string]string {
	open := map[string]string{}
	replied := map[string]bool{}
	chans, _ := d.Bus.Channels()
	for _, ch := range chans {
		msgs, _ := d.Bus.Messages(ch)
		for _, m := range msgs {
			if m.Type == model.MsgAsk {
				open[m.ID] = m.From
			}
			if m.InReplyTo != "" {
				replied[m.InReplyTo] = true
			}
		}
	}
	for id := range replied {
		delete(open, id)
	}
	return open
}

func (d *Dispatcher) isWorking(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.working[id]
}

func (d *Dispatcher) setWorking(id string, v bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v {
		d.working[id] = true
	} else {
		delete(d.working, id)
	}
}

// threadKey / splitThreadKey compose and split the (agent, thread) pending key,
// keeping each task thread's held payload separate (so resume targets the right
// session). Bus channel ids never contain the NUL separator.
func threadKey(agent, thread string) string { return agent + "\x00" + thread }

func splitThreadKey(ck string) (agent, thread string) {
	if i := strings.IndexByte(ck, 0); i >= 0 {
		return ck[:i], ck[i+1:]
	}
	return ck, ""
}

// snapshotWorking returns a fresh copy of the live working set (Tick replaces
// the policy's Working with this each cycle so finished runners free up).
func (d *Dispatcher) snapshotWorking() map[string]bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	m := make(map[string]bool, len(d.working))
	for k, v := range d.working {
		m[k] = v
	}
	return m
}

func (d *Dispatcher) takePending(agent string) []model.Message {
	d.mu.Lock()
	defer d.mu.Unlock()
	p := d.pending[agent]
	delete(d.pending, agent)
	return p
}

func (d *Dispatcher) holdPending(agent string, payload []model.Message) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pending[agent] = payload
}

// requeue prepends a failed delivery's payload so the next tick retries it.
func (d *Dispatcher) requeue(agent string, payload []model.Message) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pending[agent] = append(append([]model.Message{}, payload...), d.pending[agent]...)
}

// pendingKeys returns the (agent, thread) keys with buffered payloads, sorted.
func (d *Dispatcher) pendingKeys() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, 0, len(d.pending))
	for k := range d.pending {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (d *Dispatcher) cursorFile() string { return filepath.Join(d.Bus.L.Run(), "daemon.cursor") }

// loadCursor restores the processed-cursor; a missing/invalid file is the zero
// time (process the whole backlog once, as on a fresh workspace).
func (d *Dispatcher) loadCursor() time.Time {
	raw, err := os.ReadFile(d.cursorFile())
	if err != nil {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(raw)))
	return t
}

func (d *Dispatcher) saveCursor() {
	if d.cursor.IsZero() {
		return
	}
	_ = os.MkdirAll(d.Bus.L.Run(), 0o755)
	_ = os.WriteFile(d.cursorFile(), []byte(d.cursor.Format(time.RFC3339Nano)), 0o644)
}
