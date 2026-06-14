package wf

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func TestEscalatePostsAndTargets(t *testing.T) {
	l, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	// default target is the first conductor (claude-code in the template).
	if got := EscalateTarget(ws); got != "claude-code" {
		t.Fatalf("escalate target = %q, want claude-code", got)
	}
	if err := Escalate(ws, store.NewLedger(l), "T-001", "codex", "stuck"); err != nil {
		t.Fatal(err)
	}
	msgs, _ := bus.New(l).Messages("#escalations")
	if len(msgs) != 1 {
		t.Fatalf("want 1 escalation message, got %d", len(msgs))
	}
}
