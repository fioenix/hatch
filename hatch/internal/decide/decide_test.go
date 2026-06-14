package decide

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func TestRecordWritesADR(t *testing.T) {
	l, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	e, err := Record(ws, "meet-1", "Dùng CSV streaming", "claude-code", "Chốt: streaming + BOM UTF-8")
	if err != nil {
		t.Fatal(err)
	}
	if e.Type != model.KBDecision || e.Status != "accepted" {
		t.Fatalf("bad entry: %+v", e)
	}
	entries, _ := store.NewKB(l).List()
	found := false
	for _, x := range entries {
		if x.ID == e.ID && x.Type == model.KBDecision {
			found = true
		}
	}
	if !found {
		t.Fatal("ADR not persisted in kb/decisions")
	}
}
