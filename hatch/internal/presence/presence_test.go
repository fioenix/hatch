package presence

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

func TestPresenceDefaultsAvailableAndPersists(t *testing.T) {
	l := paths.At(t.TempDir())
	b := Load(l)
	if b.StatusOf("codex") != Available {
		t.Fatal("missing agent should be available")
	}
	if !b.CanTakeWork("codex") {
		t.Fatal("available agent can take work")
	}
	b.Set("codex", Offline, "PTO")
	if err := b.Save(l); err != nil {
		t.Fatal(err)
	}
	b2 := Load(l)
	if b2.StatusOf("codex") != Offline || b2.CanTakeWork("codex") {
		t.Fatalf("offline not persisted/honored: %+v", b2["codex"])
	}
	if b2["codex"].Note != "PTO" {
		t.Fatalf("note lost: %q", b2["codex"].Note)
	}
}
