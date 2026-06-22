// Package wake is the delivery layer's decision core: given the room roster, a
// batch of new chat messages, and the current delivery state, it decides which
// teammates to wake, which wakes to defer, and when to escalate to the boss.
//
// It is a pure function with no IO — the daemon (an adapter) tails the bus and
// spawns agents around it. Keeping the policy pure makes the seven coordination
// rules (see Decide) table-testable, which is where the real complexity lives.
//
// Hard invariant: a wake is ALWAYS the consequence of a message someone sent.
// This package never invents work; it only routes and paces conversation.
package wake

import (
	"sort"
	"strings"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

// Config tunes the delivery policy. Zero values fall back to sane defaults via
// withDefaults, so callers can pass Config{} for the standard behaviour.
type Config struct {
	Depth      int           // max auto-wake cascade depth per episode (Rule 5)
	RateCap    int           // max wakes per member per RateWindow (Rule 6)
	RateWindow time.Duration // sliding window for the rate cap
	LoopRounds int           // max agent↔agent ping-pong rounds without progress (Rule 7)
}

func (c Config) withDefaults() Config {
	if c.Depth <= 0 {
		c.Depth = 4
	}
	if c.RateCap <= 0 {
		c.RateCap = 6
	}
	if c.RateWindow <= 0 {
		c.RateWindow = time.Minute
	}
	if c.LoopRounds <= 0 {
		c.LoopRounds = 3
	}
	return c
}

// State is the delivery layer's memory between batches. The daemon persists and
// reloads it; Decide treats it as immutable input and returns the next State.
type State struct {
	Working  map[string]bool        // members mid-turn (Rule 3 debounce)
	Rate     map[string][]time.Time // recent delivered wakes per member (Rule 6)
	Depth    map[string]int         // cascade depth per episode (Rule 5)
	PingPong map[string]int         // rounds per "episode|a|b" (Rule 7)
	LastTrig map[string]string      // last wake-causing sender per episode (Rule 7)
	OpenAsks map[string]string      // open ask message id → owner member id (Rule 1b)
	Progress map[string]bool        // episodes that saw a decision/deliverable this run
}

func (s State) clone() State {
	n := State{
		Working:  map[string]bool{},
		Rate:     map[string][]time.Time{},
		Depth:    map[string]int{},
		PingPong: map[string]int{},
		LastTrig: map[string]string{},
		OpenAsks: map[string]string{},
		Progress: map[string]bool{},
	}
	for k, v := range s.Working {
		n.Working[k] = v
	}
	for k, v := range s.Rate {
		n.Rate[k] = append([]time.Time(nil), v...)
	}
	for k, v := range s.Depth {
		n.Depth[k] = v
	}
	for k, v := range s.PingPong {
		n.PingPong[k] = v
	}
	for k, v := range s.LastTrig {
		n.LastTrig[k] = v
	}
	for k, v := range s.OpenAsks {
		n.OpenAsks[k] = v
	}
	for k, v := range s.Progress {
		n.Progress[k] = v
	}
	return n
}

// Decide applies the seven delivery rules to a batch of new messages.
//
//  1. TRIGGER — only an @mention/recipient, a reply to the member's open ask,
//     or a DM wakes a member. An unaddressed message wakes no one.
//  2. COALESCE — several triggering messages for one member merge into one wake.
//  3. DEBOUNCE — a member mid-turn is not woken; its wake is held (HoldWorking).
//  4. NO SELF/ECHO — a member never wakes itself; replies don't re-wake the last
//     speaker unless it is explicitly addressed (falls out of Rule 1).
//  5. DEPTH — auto-wake cascades from agents are capped per episode; over the
//     cap escalates instead of waking. A boss message resets the episode depth.
//  6. RATE — per-member wake cap over a window; excess is held (HoldRate).
//  7. LOOP — two agents ping-ponging without progress escalate to the boss.
//
// now is injected for deterministic tests. The returned State must be fed back
// into the next call.
func Decide(roster model.Roster, msgs []model.Message, st State, cfg Config, now time.Time) (decisions []model.WakeDecision, escalations []model.Escalation, next State) {
	cfg = cfg.withDefaults()
	next = st.clone()

	wakes := map[string]*model.WakeDecision{}
	escSeen := map[string]bool{} // dedupe escalations by episode|cause

	addEsc := func(episode, cause, note string) {
		key := episode + "|" + cause
		if escSeen[key] {
			return
		}
		escSeen[key] = true
		escalations = append(escalations, model.Escalation{Episode: episode, Cause: cause, To: bossID(roster), Note: note})
	}

	for _, m := range msgs {
		e := episode(m)
		fromHuman := roster.IsHuman(m.From)

		// A boss message starts a fresh episode: reset cascade + loop memory.
		if fromHuman {
			next.Depth[e] = 0
			resetEpisodeLoops(&next, e)
		}
		// A recorded decision is progress: it clears the loop counter so a
		// resolved-then-resumed thread is not mistaken for a stuck loop.
		if m.Type == model.MsgDecision {
			next.Progress[e] = true
			resetEpisodeLoops(&next, e)
		}

		targets := recipients(roster, next, m)
		if len(targets) == 0 {
			continue // Rule 1 / Rule 4: nothing addressed → no wake.
		}

		// Rule 5: depth applies only to cascades originating from an agent.
		prospectiveDepth := next.Depth[e]
		if !fromHuman {
			prospectiveDepth = next.Depth[e] + 1
		}

		for _, t := range sortedKeys(targets) {
			a := t
			reason := targets[a]
			if a == m.From || !roster.Reachable(a) {
				continue // Rule 4 + presence guard.
			}

			if !fromHuman && prospectiveDepth > cfg.Depth {
				addEsc(e, model.EscalateDepthLimit, "cascade depth exceeded; boss decision needed")
				continue
			}

			// Rule 7: ping-pong between two agents without progress.
			if !fromHuman {
				key := pairKey(e, m.From, a)
				if next.LastTrig[e] != "" && next.LastTrig[e] != m.From {
					next.PingPong[key]++
				}
				if next.PingPong[key] >= cfg.LoopRounds && !next.Progress[e] {
					addEsc(e, model.EscalateLoopBreak, "ping-pong without progress; boss decision needed")
					continue
				}
			}

			hold := model.HoldNone
			switch {
			case st.Working[a]:
				hold = model.HoldWorking // Rule 3
			case rateExceeded(next.Rate[a], cfg, now):
				hold = model.HoldRate // Rule 6
			}

			// Rule 2: coalesce per member.
			d, ok := wakes[a]
			if !ok {
				d = &model.WakeDecision{Agent: a, Reason: reason}
				wakes[a] = d
			}
			if reason == model.WakeReplyAsk {
				d.Reason = reason // a direct answer to my question outranks a generic mention
			}
			d.Payload = append(d.Payload, m)
			// The strongest hold wins (working beats rate beats none).
			if holdRank(hold) > holdRank(d.Hold) {
				d.Hold = hold
			}

			// Record a delivered (non-held) wake for depth/rate/loop accounting.
			if hold == model.HoldNone {
				next.Rate[a] = append(prune(next.Rate[a], cfg, now), now)
				if !fromHuman && prospectiveDepth > next.Depth[e] {
					next.Depth[e] = prospectiveDepth
				}
				if !fromHuman {
					next.LastTrig[e] = m.From
				}
			}
		}
	}

	for _, a := range sortedDecisionKeys(wakes) {
		decisions = append(decisions, *wakes[a])
	}
	return decisions, escalations, next
}

// episode is the thread-root id a message belongs to: its reply target if it is
// a reply, else the message itself.
func episode(m model.Message) string {
	if m.InReplyTo != "" {
		return m.InReplyTo
	}
	return m.ID
}

// recipients resolves who a message addresses, mapping each to a wake reason.
// m.To already includes @mentions (the bus merges them at post time), plus
// explicit recipients, roles, and broadcast tokens.
func recipients(r model.Roster, st State, m model.Message) map[string]model.WakeReason {
	out := map[string]model.WakeReason{}
	dm := strings.HasPrefix(m.Channel, "dm-")
	for _, to := range m.To {
		to = strings.TrimPrefix(strings.TrimSpace(to), "@")
		switch to {
		case "", m.From:
			continue
		case "*", "all":
			for id := range r {
				if r.Reachable(id) && id != m.From {
					out[id] = model.WakeMention
				}
			}
		default:
			if r.Reachable(to) {
				if dm {
					out[to] = model.WakeDM
				} else {
					out[to] = model.WakeMention
				}
				continue
			}
			// Not a member id → treat as a role and fan out to its holders.
			for _, id := range r.WithRole(to) {
				if id != m.From {
					out[id] = model.WakeMention
				}
			}
		}
	}
	// Rule 1b: a reply landing on someone's open ask wakes that owner.
	if m.InReplyTo != "" {
		if owner, ok := st.OpenAsks[m.InReplyTo]; ok && owner != m.From && r.Reachable(owner) {
			out[owner] = model.WakeReplyAsk
		}
	}
	return out
}

func rateExceeded(stamps []time.Time, cfg Config, now time.Time) bool {
	return len(prune(stamps, cfg, now)) >= cfg.RateCap
}

func prune(stamps []time.Time, cfg Config, now time.Time) []time.Time {
	cut := now.Add(-cfg.RateWindow)
	out := stamps[:0:0]
	for _, t := range stamps {
		if t.After(cut) {
			out = append(out, t)
		}
	}
	return out
}

func resetEpisodeLoops(st *State, e string) {
	for k := range st.PingPong {
		if strings.HasPrefix(k, e+"|") {
			delete(st.PingPong, k)
		}
	}
	delete(st.LastTrig, e)
}

func pairKey(episode, a, b string) string {
	if a > b {
		a, b = b, a
	}
	return episode + "|" + a + "|" + b
}

func bossID(r model.Roster) string {
	for id, m := range r {
		if m.Kind == model.KindUser {
			return id
		}
	}
	return "user"
}

func holdRank(h model.HoldReason) int {
	switch h {
	case model.HoldWorking:
		return 2
	case model.HoldRate:
		return 1
	default:
		return 0
	}
}

func sortedKeys(m map[string]model.WakeReason) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedDecisionKeys(m map[string]*model.WakeDecision) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
