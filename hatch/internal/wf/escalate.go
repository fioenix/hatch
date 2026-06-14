package wf

import (
	"fmt"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/oncall"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// escalationThreshold is the number of gate failures on a ticket that triggers
// an automatic escalation (like a teammate flagging a senior after retries).
const escalationThreshold = 2

// EscalateTarget resolves who escalations go to: the current on-call, else
// registry policy, else the first conductor, else a human lead.
func EscalateTarget(ws *config.Workspace) string {
	if oc := oncall.Load(ws.Layout).Now(); oc != "" {
		return oc
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
func Escalate(ws *config.Workspace, lg *store.Ledger, ticket, from, why string) error {
	target := EscalateTarget(ws)
	if from == "" {
		from = "orchestrator"
	}
	if err := lg.Append(model.Entry{
		Agent: from, Ticket: ticket, Action: model.ActEscalate,
		ToRole: target, Why: why,
	}); err != nil {
		return err
	}
	_, err := bus.New(ws.Layout).Post(bus.Message{
		Channel: "#escalations", From: from, To: []string{target},
		Type: bus.TypeMsg, Body: fmt.Sprintf("@%s ESCALATION %s: %s", target, ticket, why),
	})
	return err
}

// maybeEscalate auto-escalates when a ticket has hit the gate-failure threshold
// and hasn't already been escalated.
func maybeEscalate(ws *config.Workspace, lg *store.Ledger, ticket string) {
	entries, err := lg.Recent(1)
	if err != nil {
		return
	}
	fails, escalated := 0, false
	for _, e := range entries {
		if e.Ticket != ticket {
			continue
		}
		if e.Action == model.ActGate && strings.HasPrefix(e.Result, "failed") {
			fails++
		}
		if e.Action == model.ActEscalate {
			escalated = true
		}
	}
	if fails >= escalationThreshold && !escalated {
		_ = Escalate(ws, lg, ticket, "orchestrator",
			fmt.Sprintf("gate failed %d times — cần can thiệp", fails))
	}
}
