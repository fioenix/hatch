package gate

import (
	"errors"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

// fakeRunner is a test adapter for the Runner port — no real shell.
type fakeRunner struct {
	out string
	err error
}

func (f fakeRunner) Run(string, string) (string, error) { return f.out, f.err }

func wsWithGate(g model.Gate) *config.Workspace {
	return &config.Workspace{Workflow: &model.Workflow{Gates: map[string]model.Gate{"g": g}}}
}

func TestCommandGatePassAndFailViaPort(t *testing.T) {
	pass := Evaluator{Runner: fakeRunner{out: "ok"}}.
		Evaluate(wsWithGate(model.Gate{Type: model.GateCommand, Run: "make test"}), "g", model.Ticket{}, ".")
	if !pass.Passed {
		t.Fatalf("expected pass, got %+v", pass)
	}
	fail := Evaluator{Runner: fakeRunner{out: "boom", err: errors.New("exit 1")}}.
		Evaluate(wsWithGate(model.Gate{Type: model.GateCommand, Run: "make test"}), "g", model.Ticket{}, ".")
	if fail.Passed {
		t.Fatalf("expected fail, got %+v", fail)
	}
}
