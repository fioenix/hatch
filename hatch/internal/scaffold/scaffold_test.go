package scaffold_test

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
)

// Every shipped workflow template must scaffold into a workspace that passes
// config validation (lanes/transitions/gates/roles all consistent).
func TestAllWorkflowTemplatesValidate(t *testing.T) {
	for _, wf := range scaffold.WorkflowTemplates {
		t.Run(wf, func(t *testing.T) {
			dir := t.TempDir()
			l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: wf})
			if err != nil {
				t.Fatalf("init %s: %v", wf, err)
			}
			ws, err := config.Load(l)
			if err != nil {
				t.Fatalf("load %s: %v", wf, err)
			}
			if probs := ws.Validate(); len(probs) != 0 {
				for _, p := range probs {
					t.Errorf("%s: %s", wf, p)
				}
			}
		})
	}
}

func TestUnknownWorkflowRejected(t *testing.T) {
	if _, _, err := scaffold.Init(scaffold.Options{Dir: t.TempDir(), Workflow: "nope"}); err == nil {
		t.Fatal("expected error for unknown workflow template")
	}
}
