//go:build hatch_legacy

package wf

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/oncall"
	"github.com/fioenix/overclaud/hatch/internal/paths"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func engine(l paths.Layout) Engine {
	return Engine{
		Board:  store.NewBoard(l),
		Ledger: store.NewLedger(l),
		Bus:    bus.New(l),
		OnCall: oncall.Service{L: l},
	}
}

func TestEscalatePostsAndTargets(t *testing.T) {
	l, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	eng := engine(l)
	// default target is the first conductor (claude-code in the template).
	if got := eng.EscalateTarget(ws); got != "claude-code" {
		t.Fatalf("escalate target = %q, want claude-code", got)
	}
	if err := eng.Escalate(ws, "T-001", "codex", "stuck"); err != nil {
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
	if got := engine(l).EscalateTarget(ws); got != "gemini" {
		t.Fatalf("on-call should win escalation target, got %q", got)
	}
}

func TestEscalateClimbsOrgChart(t *testing.T) {
	l, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	for i := range ws.Registry.Roles {
		if ws.Registry.Roles[i].ID == "implementer" {
			ws.Registry.Roles[i].ReportsTo = "conductor"
		}
	}
	if got := engine(l).EscalateTargetForRole(ws, "implementer"); got != "claude-code" {
		t.Fatalf("escalation should climb to conductor agent claude-code, got %q", got)
	}
}
