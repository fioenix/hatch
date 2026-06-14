package ceremony

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

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

	rep, err := Standup(w, 1)
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

	r, err := Retro(w)
	if err != nil {
		t.Fatal(err)
	}
	if r.Done != 1 || r.GateFailures != 1 || r.Blocks != 1 {
		t.Fatalf("retro counts wrong: %+v", r)
	}
}
