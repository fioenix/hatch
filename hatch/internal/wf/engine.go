// Package wf is the workflow engine: it authorises and performs lane
// transitions, enforcing gates and policies and recording the ledger.
package wf

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/gate"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// MoveOptions parameterise a transition.
type MoveOptions struct {
	To            string // target lane
	ByRole        string // role performing the move (must satisfy transition.By)
	Agent         string // agent performing the move
	Why           string // ledger reason (required)
	Handoff       string // appended to handoff notes; required for handoff action
	HumanApproved bool   // operator acknowledges human/checklist gates
	SkipGates     bool   // bypass gate evaluation (records it in the ledger)
}

// Result reports a completed move.
type Result struct {
	Ticket   model.Ticket
	From     string
	To       string
	Action   string
	Outcomes []gate.Outcome
}

// Move validates and performs a transition for ticketID.
func Move(ws *config.Workspace, b *store.Board, lg *store.Ledger, ticketID string, opt MoveOptions) (*Result, error) {
	if strings.TrimSpace(opt.Why) == "" {
		return nil, fmt.Errorf("a reason (--why) is required")
	}
	t, ok, err := b.Find(ticketID, ws.Workflow.LaneIDs())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("ticket %s not found", ticketID)
	}
	from := t.Lane
	if from == opt.To {
		return nil, fmt.Errorf("ticket %s already in %s", ticketID, opt.To)
	}
	if !ws.Workflow.HasLane(opt.To) {
		return nil, fmt.Errorf("unknown lane %q", opt.To)
	}
	tr, ok := ws.Workflow.FindTransition(from, opt.To)
	if !ok {
		return nil, fmt.Errorf("no transition %s → %s defined in workflow", from, opt.To)
	}
	if opt.ByRole != "" && !roleAllowed(tr.By, opt.ByRole) {
		return nil, fmt.Errorf("role %q may not perform %s → %s (allowed: %s)",
			opt.ByRole, from, opt.To, strings.Join(tr.By, ", "))
	}

	// depends_on must be done before claiming into an active lane.
	if tr.Action == model.ActClaim {
		if err := checkDependencies(ws, b, t); err != nil {
			return nil, err
		}
	}

	// no-self-review: a reviewer transition may not be done by the agent that
	// last held (implemented) the ticket.
	if ws.Registry.Policy.NoSelfReview && isReviewerTransition(tr) && opt.Agent != "" && opt.Agent == t.Assignee {
		return nil, fmt.Errorf("no-self-review: agent %q implemented this ticket and cannot review it", opt.Agent)
	}

	// Evaluate gates.
	var outcomes []gate.Outcome
	if !opt.SkipGates && len(tr.Gates) > 0 {
		outcomes = gate.EvaluateAll(ws, tr.Gates, t, ws.Layout.RepoRoot())
		for _, o := range outcomes {
			if o.Human && !opt.HumanApproved {
				_ = lg.Append(gateEntry(opt, t, from, "blocked", "human gate "+o.Name+" needs approval"))
				return nil, fmt.Errorf("gate %q needs a human (%s); re-run with --approve once satisfied", o.Name, o.Detail)
			}
			if !o.Human && !o.Passed {
				_ = lg.Append(gateEntry(opt, t, from, "failed", "gate "+o.Name+" failed: "+o.Detail))
				maybeEscalate(ws, lg, t.ID)
				return nil, fmt.Errorf("gate %q failed: %s", o.Name, o.Detail)
			}
		}
	}

	// Apply state changes.
	now := time.Now().Format(time.RFC3339)
	oldPath := b.Path(t)
	t.Lane = opt.To
	t.Status = opt.To
	t.Updated = now
	action := tr.Action
	if action == "" {
		action = model.ActProgress
	}
	switch action {
	case model.ActClaim:
		t.Assignee = opt.Agent
		t.Claim = &model.Claim{Agent: opt.Agent, TS: now}
		if opt.ByRole != "" {
			t.Role = opt.ByRole
		}
	}
	if opt.Handoff != "" {
		t.Body = appendHandoff(t.Body, now, opt.Agent, opt.To, opt.Handoff)
	}

	newPath, err := b.Write(t)
	if err != nil {
		return nil, err
	}
	if newPath != oldPath {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Record the ledger.
	entry := model.Entry{
		TS:      now,
		Agent:   agentOrSystem(opt.Agent),
		Ticket:  t.ID,
		Action:  action,
		From:    fmt.Sprintf("%s/ → %s/", from, opt.To),
		Why:     opt.Why,
		Branch:  t.Branch,
		Handoff: opt.Handoff,
	}
	if opt.SkipGates && len(tr.Gates) > 0 {
		entry.Note = "gates skipped"
	}
	if err := lg.Append(entry); err != nil {
		return nil, err
	}

	return &Result{Ticket: t, From: from, To: opt.To, Action: action, Outcomes: outcomes}, nil
}

func checkDependencies(ws *config.Workspace, b *store.Board, t model.Ticket) error {
	if len(t.DependsOn) == 0 {
		return nil
	}
	doneLanes := terminalLanes(ws)
	for _, dep := range t.DependsOn {
		dt, ok, err := b.Find(dep, ws.Workflow.LaneIDs())
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("dependency %s not found", dep)
		}
		if !doneLanes[dt.Lane] {
			return fmt.Errorf("dependency %s not done (in %s)", dep, dt.Lane)
		}
	}
	return nil
}

// terminalLanes are lanes with no outgoing transition (done-like).
func terminalLanes(ws *config.Workspace) map[string]bool {
	outgoing := map[string]bool{}
	for _, tr := range ws.Workflow.Transitions {
		if tr.From != "*" {
			outgoing[tr.From] = true
		}
	}
	term := map[string]bool{}
	for _, l := range ws.Workflow.Lanes {
		if !l.Side && !outgoing[l.ID] {
			term[l.ID] = true
		}
	}
	return term
}

func isReviewerTransition(tr model.Transition) bool {
	for _, r := range tr.By {
		if r == "reviewer" {
			return true
		}
	}
	return false
}

func roleAllowed(by []string, role string) bool {
	for _, r := range by {
		if r == "*" || r == role {
			return true
		}
	}
	return false
}

func agentOrSystem(a string) string {
	if a == "" {
		return "human:operator"
	}
	return a
}

func gateEntry(opt MoveOptions, t model.Ticket, from, result, why string) model.Entry {
	return model.Entry{
		Agent:  agentOrSystem(opt.Agent),
		Ticket: t.ID,
		Action: model.ActGate,
		From:   fmt.Sprintf("%s/ → %s/", from, opt.To),
		Result: result,
		Why:    why,
	}
}

// appendHandoff inserts a dated bullet under the ticket's "Handoff notes".
func appendHandoff(body, ts, agent, lane, note string) string {
	bullet := fmt.Sprintf("- %s %s→%s: %s", strings.SplitN(ts, "T", 2)[0], agent, lane, note)
	if strings.Contains(body, "## Handoff notes") {
		return strings.TrimRight(body, "\n") + "\n" + bullet + "\n"
	}
	return strings.TrimRight(body, "\n") + "\n\n## Handoff notes\n" + bullet + "\n"
}
