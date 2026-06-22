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
	"sort"
	"sync"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/roster"
	"github.com/fioenix/overclaud/hatch/internal/wake"
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
	working map[string]bool            // members with an in-flight Runner.Wake
	state   wake.State                 // wake policy memory across ticks
	pending map[string][]model.Message // held payloads awaiting redelivery
	cursor  time.Time                  // newest message TS already processed
}

// New returns a Dispatcher ready to Tick.
func New(b *bus.Bus, rs *roster.Store, r Runner, cfg wake.Config) *Dispatcher {
	return &Dispatcher{
		Bus: b, Roster: rs, Runner: r, Cfg: cfg,
		working: map[string]bool{},
		state:   freshState(),
		pending: map[string][]model.Message{},
	}
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
	}

	// Snapshot the live working set and recompute open asks for the policy.
	d.mu.Lock()
	for k, v := range d.working {
		d.state.Working[k] = v
	}
	d.mu.Unlock()
	d.state.OpenAsks = d.openAsks()

	decisions, esc, next := wake.Decide(r, newMsgs, d.state, d.Cfg, now)
	d.state = next

	// Merge held payloads from earlier ticks, then dispatch or re-hold.
	for i := range decisions {
		dec := decisions[i]
		if buf := d.pending[dec.Agent]; len(buf) > 0 {
			dec.Payload = append(append([]model.Message{}, buf...), dec.Payload...)
		}
		if dec.Hold == model.HoldNone && !d.isWorking(dec.Agent) {
			delete(d.pending, dec.Agent)
			d.dispatch(r[dec.Agent], dec.Payload)
			dispatched = append(dispatched, dec)
		} else {
			d.pending[dec.Agent] = dec.Payload
		}
	}

	// Flush members that became free and still have buffered payloads.
	for _, agent := range sortedPending(d.pending) {
		if d.isWorking(agent) {
			continue
		}
		if _, queued := decisionFor(decisions, agent); queued {
			continue // already handled above this tick
		}
		buf := d.pending[agent]
		delete(d.pending, agent)
		d.dispatch(r[agent], buf)
		dispatched = append(dispatched, model.WakeDecision{Agent: agent, Reason: model.WakeMention, Payload: buf})
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
func (d *Dispatcher) dispatch(m model.Member, payload []model.Message) {
	if m.ID == "" {
		return
	}
	d.setWorking(m.ID, true)
	_ = d.Roster.SetStatus(m.ID, model.MemberOnline)
	go func() {
		defer func() {
			d.setWorking(m.ID, false)
			_ = d.Roster.SetStatus(m.ID, model.MemberSuspended)
		}()
		_ = d.Runner.Wake(m, payload)
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

func decisionFor(ds []model.WakeDecision, agent string) (model.WakeDecision, bool) {
	for _, x := range ds {
		if x.Agent == agent {
			return x, true
		}
	}
	return model.WakeDecision{}, false
}

func sortedPending(m map[string][]model.Message) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
