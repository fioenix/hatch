package roster

import (
	"testing"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return New(paths.Layout{Root: t.TempDir()})
}

func TestJoinAndLoad(t *testing.T) {
	s := newStore(t)
	if _, err := s.Join(model.Member{ID: "codex", Kind: "codex", Roles: []string{"implementer"}, SessionID: "sess-1"}); err != nil {
		t.Fatal(err)
	}
	r, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := r["codex"]
	if !ok {
		t.Fatal("codex not in roster")
	}
	if m.Status != model.MemberOnline {
		t.Fatalf("want online, got %q", m.Status)
	}
	if m.SessionID != "sess-1" || m.LastSeen == "" {
		t.Fatalf("session/last-seen not recorded: %+v", m)
	}
}

func TestJoinPreservesSession(t *testing.T) {
	s := newStore(t)
	_, _ = s.Join(model.Member{ID: "codex", Kind: "codex", SessionID: "sess-1"})
	// Re-join without a session id must keep the previously recorded one.
	_, _ = s.Join(model.Member{ID: "codex", Kind: "codex"})
	r, _ := s.Load()
	if r["codex"].SessionID != "sess-1" {
		t.Fatalf("session id should be preserved, got %q", r["codex"].SessionID)
	}
}

func TestLeaveMarksOfflineNotReachable(t *testing.T) {
	s := newStore(t)
	_, _ = s.Join(model.Member{ID: "codex", Kind: "codex"})
	if err := s.Leave("codex"); err != nil {
		t.Fatal(err)
	}
	r, _ := s.Load()
	if r["codex"].Status != model.MemberOffline {
		t.Fatalf("want offline, got %q", r["codex"].Status)
	}
	if r.Reachable("codex") {
		t.Fatal("offline member must not be reachable")
	}
}

func TestTouchRevivesIdle(t *testing.T) {
	s := newStore(t)
	_, _ = s.Join(model.Member{ID: "codex", Kind: "codex"})
	_ = s.SetStatus("codex", model.MemberIdle)
	if err := s.Touch("codex"); err != nil {
		t.Fatal(err)
	}
	r, _ := s.Load()
	if r["codex"].Status != model.MemberOnline {
		t.Fatalf("touch should revive idle→online, got %q", r["codex"].Status)
	}
}

func TestEffectiveDerivesIdle(t *testing.T) {
	s := newStore(t)
	_, _ = s.Join(model.Member{ID: "codex", Kind: "codex"})
	// Look at the roster well after the idle threshold.
	r, err := s.Effective(time.Now().Add(2 * IdleAfter))
	if err != nil {
		t.Fatal(err)
	}
	if r["codex"].Status != model.MemberIdle {
		t.Fatalf("want derived idle, got %q", r["codex"].Status)
	}
	// Effective must not persist the derivation.
	persisted, _ := s.Load()
	if persisted["codex"].Status != model.MemberOnline {
		t.Fatalf("Effective must not persist idle, got %q", persisted["codex"].Status)
	}
}
