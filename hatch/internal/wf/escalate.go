package wf

import (
	"fmt"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

// escalationThreshold is the number of gate failures on a ticket that triggers
// an automatic escalation (like a teammate flagging a senior after retries).
const escalationThreshold = 2

// EscalateTarget resolves the system-level escalation target (no ticket role).
func (e Engine) EscalateTarget(ws *config.Workspace) string {
	return e.EscalateTargetForRole(ws, "")
}

// EscalateTargetForRole resolves who an escalation goes to: the current on-call,
// else up the org chart (the manager of the ticket's role), else registry
// policy, else the first conductor, else a human lead.
func (e Engine) EscalateTargetForRole(ws *config.Workspace, role string) string {
	if e.OnCall != nil {
		if oc := e.OnCall.Current(); oc != "" {
			return oc
		}
	}
	if role != "" {
		if r, ok := ws.Registry.RoleByID(role); ok && r.ReportsTo != "" {
			if as := ws.Registry.AgentsForRole(r.ReportsTo); len(as) > 0 {
				return as[0].ID
			}
		}
	}
	if ws.Registry.Policy.EscalateTo != "" {
		return ws.Registry.Policy.EscalateTo
	}
	if as := ws.Registry.AgentsForRole("conductor"); len(as) > 0 {
		return as[0].ID
	}
	return "human:lead"
}

// Escalate records an escalation in the ledger and notifies the target on the
// #escalations channel.
func (e Engine) Escalate(ws *config.Workspace, ticket, from, why string) error {
	role := ""
	if t, ok, _ := e.Board.Find(ticket, ws.Workflow.LaneIDs()); ok {
		role = t.Role
	}
	target := e.EscalateTargetForRole(ws, role)
	if from == "" {
		from = "orchestrator"
	}
	if err := e.Ledger.Append(model.Entry{
		Agent: from, Ticket: ticket, Action: model.ActEscalate,
		ToRole: target, Why: why,
	}); err != nil {
		return err
	}
	return e.Bus.Notify("#escalations", from, []string{target},
		fmt.Sprintf("@%s ESCALATION %s: %s", target, ticket, why))
}

// maybeEscalate auto-escalates when a ticket has hit the gate-failure threshold
// and hasn't already been escalated.
func (e Engine) maybeEscalate(ws *config.Workspace, ticket string) {
	entries, err := e.Ledger.Recent(1)
	if err != nil {
		return
	}
	fails, escalated := 0, false
	for _, en := range entries {
		if en.Ticket != ticket {
			continue
		}
		if en.Action == model.ActGate && strings.HasPrefix(en.Result, "failed") {
			fails++
		}
		if en.Action == model.ActEscalate {
			escalated = true
		}
	}
	if fails >= escalationThreshold && !escalated {
		_ = e.Escalate(ws, ticket, "orchestrator",
			fmt.Sprintf("gate failed %d times — cần can thiệp", fails))
	}
}
