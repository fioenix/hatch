package wake

import (
	"testing"
	"time"

	"github.com/fioenix/hatch/internal/model"
)

func room() model.Roster {
	return model.Roster{
		"boss":   {ID: "boss", Kind: model.KindUser, Status: model.MemberOnline},
		"claude": {ID: "claude", Kind: "claude", Roles: []string{"reviewer"}, Status: model.MemberOnline},
		"codex":  {ID: "codex", Kind: "codex", Roles: []string{"implementer"}, Status: model.MemberSuspended},
		"agy":    {ID: "agy", Kind: "agy", Roles: []string{"implementer"}, Status: model.MemberSuspended},
	}
}

func freshState() State {
	return State{
		Working: map[string]bool{}, Rate: map[string][]time.Time{}, Depth: map[string]int{},
		PingPong: map[string]int{}, LastTrig: map[string]string{}, OpenAsks: map[string]string{}, Progress: map[string]bool{},
	}
}

func msg(id, from, ch string, to []string, typ, replyTo, body string) model.Message {
	return model.Message{ID: id, From: from, Channel: ch, To: to, Type: typ, InReplyTo: replyTo, Body: body}
}

func wakeOf(ds []model.WakeDecision, agent string) (model.WakeDecision, bool) {
	for _, d := range ds {
		if d.Agent == agent {
			return d, true
		}
	}
	return model.WakeDecision{}, false
}

// Rule 1: only addressed members are woken; an unaddressed broadcast wakes none.
func TestTriggerOnlyAddressed(t *testing.T) {
	now := time.Now()
	ds, _, _ := Decide(room(), []model.Message{
		msg("m1", "boss", "#dev", []string{"codex"}, model.MsgText, "", "fix the bug"),
	}, freshState(), Config{}, now)
	if len(ds) != 1 || ds[0].Agent != "codex" {
		t.Fatalf("want one wake to codex, got %+v", ds)
	}

	ds2, _, _ := Decide(room(), []model.Message{
		msg("m2", "claude", "#dev", nil, model.MsgText, "", "thinking out loud, no mention"),
	}, freshState(), Config{}, now)
	if len(ds2) != 0 {
		t.Fatalf("unaddressed message must wake no one, got %+v", ds2)
	}
}

// Rule 4: a member never wakes itself even if it appears in recipients.
func TestNoSelfWake(t *testing.T) {
	ds, _, _ := Decide(room(), []model.Message{
		msg("m1", "codex", "#dev", []string{"codex", "claude"}, model.MsgText, "", "@codex @claude"),
	}, freshState(), Config{}, time.Now())
	if _, ok := wakeOf(ds, "codex"); ok {
		t.Fatal("codex must not wake itself")
	}
	if _, ok := wakeOf(ds, "claude"); !ok {
		t.Fatal("claude should still be woken")
	}
}

// Rule 2: multiple messages to one member coalesce into a single wake.
func TestCoalesce(t *testing.T) {
	now := time.Now()
	ds, _, _ := Decide(room(), []model.Message{
		msg("m1", "boss", "#dev", []string{"codex"}, model.MsgText, "", "part one"),
		msg("m2", "claude", "#dev", []string{"codex"}, model.MsgText, "", "part two"),
	}, freshState(), Config{}, now)
	d, ok := wakeOf(ds, "codex")
	if !ok || len(d.Payload) != 2 {
		t.Fatalf("want one coalesced wake with 2 payloads, got %+v", ds)
	}
}

// Rule 3: a member mid-turn is held, not skipped.
func TestDebounceWorking(t *testing.T) {
	st := freshState()
	st.Working["codex"] = true
	ds, _, _ := Decide(room(), []model.Message{
		msg("m1", "boss", "#dev", []string{"codex"}, model.MsgText, "", "ping"),
	}, st, Config{}, time.Now())
	d, ok := wakeOf(ds, "codex")
	if !ok || d.Hold != model.HoldWorking {
		t.Fatalf("want codex held as working, got %+v", ds)
	}
}

// Rule 1b: a reply to an open ask wakes the asker with reason reply_to_open_ask.
func TestReplyToOpenAsk(t *testing.T) {
	st := freshState()
	st.OpenAsks["ask1"] = "claude"
	ds, _, _ := Decide(room(), []model.Message{
		msg("m1", "codex", "#dev", nil, model.MsgReply, "ask1", "here is the answer"),
	}, st, Config{}, time.Now())
	d, ok := wakeOf(ds, "claude")
	if !ok || d.Reason != model.WakeReplyAsk {
		t.Fatalf("want claude woken for open ask, got %+v", ds)
	}
}

// Rule 5: an agent→agent cascade past the depth cap escalates to the boss.
func TestDepthLimitEscalates(t *testing.T) {
	cfg := Config{Depth: 2}
	st := freshState()
	now := time.Now()
	var allEsc []model.Escalation
	// Simulate a chain in one episode "E": each agent message wakes the next.
	chain := []model.Message{
		msg("a", "claude", "#dev", []string{"codex"}, model.MsgText, "E", "1"),
		msg("b", "codex", "#dev", []string{"agy"}, model.MsgText, "E", "2"),
		msg("c", "agy", "#dev", []string{"claude"}, model.MsgText, "E", "3"),
	}
	for _, m := range chain {
		var esc []model.Escalation
		_, esc, st = Decide(room(), []model.Message{m}, st, cfg, now)
		allEsc = append(allEsc, esc...)
	}
	found := false
	for _, e := range allEsc {
		if e.Cause == model.EscalateDepthLimit {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a depth_limit escalation, got %+v", allEsc)
	}
}

// A boss message resets episode cascade depth (boss intervention is fresh).
func TestBossResetsDepth(t *testing.T) {
	cfg := Config{Depth: 1}
	st := freshState()
	st.Depth["E"] = 5 // pretend the cascade was deep
	now := time.Now()
	ds, esc, _ := Decide(room(), []model.Message{
		msg("m1", "boss", "#dev", []string{"codex"}, model.MsgText, "E", "fresh direction"),
	}, st, cfg, now)
	if len(esc) != 0 {
		t.Fatalf("boss message must not escalate on depth, got %+v", esc)
	}
	if _, ok := wakeOf(ds, "codex"); !ok {
		t.Fatal("boss message should wake codex despite prior deep cascade")
	}
}

// Rule 6: per-member rate cap holds excess wakes.
func TestRateCapHolds(t *testing.T) {
	cfg := Config{RateCap: 2, RateWindow: time.Minute}
	st := freshState()
	now := time.Now()
	st.Rate["codex"] = []time.Time{now.Add(-time.Second), now.Add(-2 * time.Second)} // already 2 in window
	ds, _, _ := Decide(room(), []model.Message{
		msg("m1", "boss", "#dev", []string{"codex"}, model.MsgText, "", "one more"),
	}, st, cfg, now)
	d, ok := wakeOf(ds, "codex")
	if !ok || d.Hold != model.HoldRate {
		t.Fatalf("want codex held by rate cap, got %+v", ds)
	}
}

// Role fan-out: addressing a role wakes its reachable holders.
func TestRoleFanOut(t *testing.T) {
	ds, _, _ := Decide(room(), []model.Message{
		msg("m1", "boss", "#dev", []string{"implementer"}, model.MsgText, "", "@implementer pick this up"),
	}, freshState(), Config{}, time.Now())
	if _, ok := wakeOf(ds, "codex"); !ok {
		t.Fatal("codex (implementer) should be woken")
	}
	if _, ok := wakeOf(ds, "agy"); !ok {
		t.Fatal("agy (implementer) should be woken")
	}
	if _, ok := wakeOf(ds, "claude"); ok {
		t.Fatal("claude (reviewer) should not be woken by an implementer mention")
	}
}
