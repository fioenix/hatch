package config

import (
	"strings"
	"testing"

	"github.com/fioenix/hatch/internal/model"
)

func TestValidateWorkflowDetectsUndefinedGateAndLane(t *testing.T) {
	w := &model.Workflow{
		Version: 1,
		Lanes:   []model.Lane{{ID: "backlog"}, {ID: "done"}},
		Transitions: []model.Transition{
			{From: "backlog", To: "missing", By: []string{"*"}},
			{From: "backlog", To: "done", By: []string{"reviewer"}, Gates: []string{"ghost"}},
		},
	}
	probs := ValidateWorkflow(w)
	joined := problemsString(probs)
	if !strings.Contains(joined, "missing") {
		t.Errorf("expected unknown lane problem, got: %s", joined)
	}
	if !strings.Contains(joined, "ghost") {
		t.Errorf("expected undefined gate problem, got: %s", joined)
	}
}

func TestValidateWorkflowReachability(t *testing.T) {
	// every lane has an outgoing transition → no terminal lane.
	w := &model.Workflow{
		Version: 1,
		Lanes:   []model.Lane{{ID: "a"}, {ID: "b"}},
		Transitions: []model.Transition{
			{From: "a", To: "b", By: []string{"*"}},
			{From: "b", To: "a", By: []string{"*"}},
		},
	}
	if got := problemsString(ValidateWorkflow(w)); !strings.Contains(got, "terminal") {
		t.Errorf("expected terminal-lane problem, got: %s", got)
	}
}

func TestValidateCrossRefsUnknownRole(t *testing.T) {
	r := &model.Registry{Version: 1, Roles: []model.Role{{ID: "implementer"}}}
	w := &model.Workflow{
		Version:     1,
		Lanes:       []model.Lane{{ID: "x"}, {ID: "y"}},
		Transitions: []model.Transition{{From: "x", To: "y", By: []string{"ghostrole"}}},
	}
	if got := problemsString(ValidateCrossRefs(r, w)); !strings.Contains(got, "ghostrole") {
		t.Errorf("expected unknown role problem, got: %s", got)
	}
}

func problemsString(ps []Problem) string {
	var b strings.Builder
	for _, p := range ps {
		b.WriteString(p.String())
		b.WriteString("\n")
	}
	return b.String()
}
