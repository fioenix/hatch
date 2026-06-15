// Package gate evaluates workflow gates: shell commands, checklists, required
// fields, registry policies and human approvals.
//
// The side-effecting part — running a command — sits behind the Runner port so
// it can be swapped (e.g. a fake in tests). ShellRunner is the default adapter.
package gate

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

// Runner is the port for executing a gate command. Implementations decide how
// (and whether) to actually run it.
type Runner interface {
	Run(cmdline, dir string) (output string, err error)
}

// ShellRunner is the production adapter: runs the command via `sh -c`.
type ShellRunner struct{}

func (ShellRunner) Run(cmdline, dir string) (string, error) {
	c := exec.Command("sh", "-c", cmdline)
	c.Dir = dir
	out, err := c.CombinedOutput()
	return string(out), err
}

// Evaluator evaluates gates using an injected Runner.
type Evaluator struct{ Runner Runner }

// Outcome is the result of evaluating one gate.
type Outcome struct {
	Name   string
	Type   string
	Passed bool
	Human  bool   // true if this gate needs a human (not auto-decidable)
	Detail string // explanation / command output tail
}

// Evaluate runs a single named gate for a ticket. Command gates execute in
// repoRoot via the Runner. Human gates never auto-pass; they return Human=true.
func (e Evaluator) Evaluate(ws *config.Workspace, name string, t model.Ticket, repoRoot string) Outcome {
	g, ok := ws.Workflow.Gates[name]
	if !ok {
		return Outcome{Name: name, Passed: false, Detail: "gate not defined"}
	}
	o := Outcome{Name: name, Type: g.Type}
	switch g.Type {
	case model.GateCommand:
		out, err := e.Runner.Run(g.Run, repoRoot)
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

// EvaluateAll runs the gates a transition declares.
func (e Evaluator) EvaluateAll(ws *config.Workspace, names []string, t model.Ticket, repoRoot string) []Outcome {
	outs := make([]Outcome, 0, len(names))
	for _, n := range names {
		outs = append(outs, e.Evaluate(ws, n, t, repoRoot))
	}
	return outs
}

// Default is the production evaluator (shell command runner).
var Default = Evaluator{Runner: ShellRunner{}}

// Evaluate / EvaluateAll are convenience wrappers over the Default evaluator,
// so callers that don't need a custom Runner stay terse.
func Evaluate(ws *config.Workspace, name string, t model.Ticket, repoRoot string) Outcome {
	return Default.Evaluate(ws, name, t, repoRoot)
}

func EvaluateAll(ws *config.Workspace, names []string, t model.Ticket, repoRoot string) []Outcome {
	return Default.EvaluateAll(ws, names, t, repoRoot)
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
