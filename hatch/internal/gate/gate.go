// Package gate evaluates workflow gates: shell commands, checklists, required
// fields, registry policies and human approvals.
package gate

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

// Outcome is the result of evaluating one gate.
type Outcome struct {
	Name   string
	Type   string
	Passed bool
	Human  bool   // true if this gate needs a human (not auto-decidable)
	Detail string // explanation / command output tail
}

// Evaluate runs a single named gate for a ticket. Command gates execute in
// repoRoot. Human gates never auto-pass; they return Human=true.
func Evaluate(ws *config.Workspace, name string, t model.Ticket, repoRoot string) Outcome {
	g, ok := ws.Workflow.Gates[name]
	if !ok {
		return Outcome{Name: name, Passed: false, Detail: "gate not defined"}
	}
	o := Outcome{Name: name, Type: g.Type}
	switch g.Type {
	case model.GateCommand:
		out, err := runShell(g.Run, repoRoot)
		o.Passed = err == nil
		o.Detail = tail(out, 400)
		if err != nil && o.Detail == "" {
			o.Detail = err.Error()
		}
	case model.GateRequired:
		val := ticketField(t, g.Field)
		o.Passed = strings.TrimSpace(val) != ""
		if !o.Passed {
			o.Detail = fmt.Sprintf("required field %q is empty", g.Field)
		}
	case model.GatePolicy:
		o.Passed, o.Detail = checkPolicy(ws.Registry.Policy, g.Ref)
	case model.GateChecklist:
		// Checklists are human-judged; surface the reference for the operator.
		o.Human = true
		o.Detail = "checklist: " + g.Ref
	case model.GateHuman:
		o.Human = true
		o.Detail = "awaiting human approval"
	default:
		o.Detail = "unknown gate type " + g.Type
	}
	return o
}

// EvaluateAll runs the gates a transition declares and returns the outcomes.
func EvaluateAll(ws *config.Workspace, names []string, t model.Ticket, repoRoot string) []Outcome {
	outs := make([]Outcome, 0, len(names))
	for _, n := range names {
		outs = append(outs, Evaluate(ws, n, t, repoRoot))
	}
	return outs
}

func runShell(cmdline, dir string) (string, error) {
	c := exec.Command("sh", "-c", cmdline)
	c.Dir = dir
	out, err := c.CombinedOutput()
	return string(out), err
}

func ticketField(t model.Ticket, field string) string {
	switch field {
	case "handoff":
		// handoff lives in the ticket body under "Handoff notes"; treat the
		// presence of that section's content as the field.
		return extractSection(t.Body, "Handoff notes")
	case "branch":
		return t.Branch
	case "assignee":
		return t.Assignee
	default:
		return ""
	}
}

// extractSection returns the text following a "## <title>" heading.
func extractSection(body, title string) string {
	lines := strings.Split(body, "\n")
	var buf []string
	in := false
	for _, ln := range lines {
		if strings.HasPrefix(ln, "## ") {
			if in {
				break
			}
			if strings.Contains(strings.ToLower(ln), strings.ToLower(title)) {
				in = true
			}
			continue
		}
		if in && strings.TrimSpace(ln) != "" && !strings.HasPrefix(strings.TrimSpace(ln), "<!--") {
			buf = append(buf, ln)
		}
	}
	return strings.TrimSpace(strings.Join(buf, "\n"))
}

func checkPolicy(p model.Policy, ref string) (bool, string) {
	switch ref {
	case "no_self_review":
		if p.NoSelfReview {
			return true, "policy no_self_review enabled (enforced at move time)"
		}
		return true, "policy no_self_review disabled"
	case "human_merge":
		return p.HumanMerge, "human_merge policy"
	default:
		return true, "unknown policy ref " + ref
	}
}

func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
