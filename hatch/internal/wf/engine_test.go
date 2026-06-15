package wf

import (
	"testing"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/oncall"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func setup(t *testing.T) (*config.Workspace, Engine, *store.Board) {
	t.Helper()
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := config.Load(l)
	if err != nil {
		t.Fatal(err)
	}
	eng := Engine{Board: store.NewBoard(l), Ledger: store.NewLedger(l), Bus: bus.New(l), OnCall: oncall.Service{L: l}}
	return ws, eng, store.NewBoard(l)
}

func writeTicket(t *testing.T, b *store.Board, tk model.Ticket) {
	t.Helper()
	tk.Lane = "backlog"
	tk.Status = "backlog"
	tk.Created = time.Now().Format(time.RFC3339)
	if _, err := b.Write(tk); err != nil {
		t.Fatal(err)
	}
}

func TestClaimBlockedByDependency(t *testing.T) {
	ws, eng, b := setup(t)
	writeTicket(t, b, model.Ticket{ID: "T-001", Title: "dep", Role: "implementer"})
	writeTicket(t, b, model.Ticket{ID: "T-002", Title: "main", Role: "implementer", DependsOn: []string{"T-001"}})

	_, err := eng.Move(ws, "T-002", MoveOptions{To: "in-progress", ByRole: "implementer", Agent: "codex", Why: "go"})
	if err == nil {
		t.Fatal("expected claim to be blocked by unfinished dependency")
	}
}

func TestClaimSetsAssignee(t *testing.T) {
	ws, eng, b := setup(t)
	writeTicket(t, b, model.Ticket{ID: "T-001", Title: "x", Role: "implementer"})

	res, err := eng.Move(ws, "T-001", MoveOptions{To: "in-progress", ByRole: "implementer", Agent: "codex", Why: "claim"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ticket.Assignee != "codex" || res.Ticket.Claim == nil {
		t.Fatalf("claim did not set assignee/lock: %+v", res.Ticket)
	}
	if res.Action != model.ActClaim {
		t.Fatalf("expected claim action, got %s", res.Action)
	}
}

func TestRoleNotAllowed(t *testing.T) {
	ws, eng, b := setup(t)
	writeTicket(t, b, model.Ticket{ID: "T-001", Title: "x", Role: "reviewer"})
	// reviewer is not in the claim transition's `by` list.
	_, err := eng.Move(ws, "T-001", MoveOptions{To: "in-progress", ByRole: "reviewer", Agent: "claude-code", Why: "x"})
	if err == nil {
		t.Fatal("expected role-not-allowed error")
	}
}

func TestNoSelfReview(t *testing.T) {
	ws, eng, b := setup(t)
	writeTicket(t, b, model.Ticket{ID: "T-001", Title: "x", Role: "implementer"})
	if _, err := eng.Move(ws, "T-001", MoveOptions{To: "in-progress", ByRole: "implementer", Agent: "codex", Why: "claim"}); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Move(ws, "T-001", MoveOptions{To: "review", ByRole: "implementer", Agent: "codex", Why: "done", Handoff: "h", SkipGates: true}); err != nil {
		t.Fatal(err)
	}
	// codex implemented it; it must not be able to review it.
	_, err := eng.Move(ws, "T-001", MoveOptions{To: "done", ByRole: "reviewer", Agent: "codex", Why: "lgtm", SkipGates: true})
	if err == nil {
		t.Fatal("expected no-self-review to block")
	}
	// a different agent can.
	if _, err := eng.Move(ws, "T-001", MoveOptions{To: "done", ByRole: "reviewer", Agent: "claude-code", Why: "approved", SkipGates: true}); err != nil {
		t.Fatalf("reviewer move failed: %v", err)
	}
}

func TestWhyRequired(t *testing.T) {
	ws, eng, b := setup(t)
	writeTicket(t, b, model.Ticket{ID: "T-001", Title: "x", Role: "implementer"})
	if _, err := eng.Move(ws, "T-001", MoveOptions{To: "in-progress", ByRole: "implementer", Agent: "codex"}); err == nil {
		t.Fatal("expected error when why is empty")
	}
}
