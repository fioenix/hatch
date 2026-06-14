package wf

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/oncall"
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

func TestEscalateTargetPrefersOncall(t *testing.T) {
	l, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	(oncall.Rotation{Order: []string{"gemini"}}).Save(l)
	if got := EscalateTarget(ws); got != "gemini" {
		t.Fatalf("on-call should win escalation target, got %q", got)
	}
}

func TestEscalateClimbsOrgChart(t *testing.T) {
	l, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	// implementer reports to conductor (claude-code) in the default registry?
	// set it explicitly for the test.
	for i := range ws.Registry.Roles {
		if ws.Registry.Roles[i].ID == "implementer" {
			ws.Registry.Roles[i].ReportsTo = "conductor"
		}
	}
	if got := EscalateTargetForRole(ws, "implementer"); got != "claude-code" {
		t.Fatalf("escalation should climb to conductor agent claude-code, got %q", got)
	}
}
