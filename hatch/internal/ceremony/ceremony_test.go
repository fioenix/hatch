package ceremony

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func svc(w *config.Workspace) Service {
	return Service{
		Board:  store.NewBoard(w.Layout),
		Ledger: store.NewLedger(w.Layout),
		Bus:    bus.New(w.Layout),
		KB:     store.NewKB(w.Layout),
	}
}

func ws(t *testing.T) *config.Workspace {
	t.Helper()
	l, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	w, err := config.Load(l)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestStandupDigestsByAgent(t *testing.T) {
	w := ws(t)
	lg := store.NewLedger(w.Layout)
	lg.Append(model.Entry{Agent: "codex", Ticket: "T-001", Action: model.ActClaim, Why: "x"})
	lg.Append(model.Entry{Agent: "codex", Ticket: "T-001", Action: model.ActHandoff, Why: "y", Handoff: "done"})
	lg.Append(model.Entry{Agent: "claude-code", Ticket: "T-001", Action: model.ActReview, Why: "z", Result: "approved"})

	rep, err := svc(w).Standup(w, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rep.PerAgent["codex"]; !ok {
		t.Fatalf("expected codex in digest: %v", rep.PerAgent)
	}
	if !strings.Contains(rep.Markdown, "claude-code") {
		t.Errorf("report missing claude-code:\n%s", rep.Markdown)
	}
}

func TestRetroCounts(t *testing.T) {
	w := ws(t)
	lg := store.NewLedger(w.Layout)
	lg.Append(model.Entry{Agent: "codex", Ticket: "T-1", Action: model.ActGate, Result: "failed: x", Why: "g"})
	lg.Append(model.Entry{Agent: "claude-code", Ticket: "T-1", Action: model.ActDone, Why: "merged"})
	lg.Append(model.Entry{Agent: "codex", Ticket: "T-2", Action: model.ActBlock, Why: "stuck"})

	r, err := svc(w).Retro(w)
	if err != nil {
		t.Fatal(err)
	}
	if r.Done != 1 || r.GateFailures != 1 || r.Blocks != 1 {
		t.Fatalf("retro counts wrong: %+v", r)
	}
}

func TestDemoAndGrooming(t *testing.T) {
	w := ws(t)
	// grooming: a fresh backlog ticket from `ticket new` has TODO body + no priority.
	b := store.NewBoard(w.Layout)
	b.Write(model.Ticket{ID: "T-001", Lane: "backlog", Status: "backlog", Title: "vague", Body: "## Acceptance\nTODO\n"})
	_, items, err := svc(w).Grooming(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 groom item, got %d", len(items))
	}
	// demo: a ticket in the terminal lane shows up.
	b.Write(model.Ticket{ID: "T-002", Lane: "done", Status: "done", Title: "shipped", Assignee: "codex"})
	_, shown, err := svc(w).Demo(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(shown) != 1 || shown[0].ID != "T-002" {
		t.Fatalf("demo should show T-002, got %+v", shown)
	}
}
