package session

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return New(paths.At(t.TempDir()))
}

func TestPutGetRoundTrip(t *testing.T) {
	s := newStore(t)
	if _, ok := s.Get("codex", "t-1"); ok {
		t.Fatal("empty store should miss")
	}
	in := model.Session{Agent: "codex", Thread: "t-1", Kind: "codex", ID: "uuid-1", Status: model.SessionLive}
	if err := s.Put(in); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get("codex", "t-1")
	if !ok || got.ID != "uuid-1" || got.Status != model.SessionLive {
		t.Fatalf("round-trip wrong: %+v ok=%v", got, ok)
	}
	// a different thread is a different session
	if _, ok := s.Get("codex", "t-2"); ok {
		t.Fatal("t-2 should be independent of t-1")
	}
}

func TestPutSupersedePushesHistory(t *testing.T) {
	s := newStore(t)
	_ = s.Put(model.Session{Agent: "claude-code", Thread: "t-1", ID: "old", Status: model.SessionLive})
	_ = s.Put(model.Session{Agent: "claude-code", Thread: "t-1", ID: "new", Status: model.SessionLive})
	got, _ := s.Get("claude-code", "t-1")
	if got.ID != "new" {
		t.Fatalf("want current id new, got %q", got.ID)
	}
	if len(got.History) != 1 || got.History[0] != "old" {
		t.Fatalf("want history [old], got %v", got.History)
	}
}

func TestMarkStale(t *testing.T) {
	s := newStore(t)
	_ = s.Put(model.Session{Agent: "codex", Thread: "t-1", ID: "x", Status: model.SessionLive})
	if err := s.MarkStale("codex", "t-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("codex", "t-1")
	if got.Status != model.SessionStale {
		t.Fatalf("want stale, got %q", got.Status)
	}
	// marking a missing session is a no-op, not an error
	if err := s.MarkStale("nobody", "t-9"); err != nil {
		t.Fatal(err)
	}
}

func TestAllSorted(t *testing.T) {
	s := newStore(t)
	_ = s.Put(model.Session{Agent: "codex", Thread: "t-2", ID: "b"})
	_ = s.Put(model.Session{Agent: "codex", Thread: "t-1", ID: "a"})
	_ = s.Put(model.Session{Agent: "claude-code", Thread: "t-9", ID: "c"})
	all := s.All()
	if len(all) != 3 {
		t.Fatalf("want 3, got %d", len(all))
	}
	if all[0].Agent != "claude-code" || all[1].Thread != "t-1" || all[2].Thread != "t-2" {
		t.Fatalf("not sorted by agent then thread: %+v", all)
	}
}
