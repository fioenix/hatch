package cli

import (
	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/oncall"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
	"github.com/fioenix/overclaud/hatch/internal/store"
	"github.com/fioenix/overclaud/hatch/internal/wf"
)

// orch is the composition root for the orchestrator: it wires the ledger + bus
// adapters into the port-based Orchestrator.
func orch(ws *config.Workspace) orchestrator.Orchestrator {
	return orchestrator.Orchestrator{
		Ledger: store.NewLedger(ws.Layout),
		Bus:    bus.New(ws.Layout),
	}
}

// engineFor is the composition root for the workflow engine: it wires the
// concrete filesystem/bus/on-call adapters into the port-based wf.Engine.
func engineFor(ws *config.Workspace) wf.Engine {
	return wf.Engine{
		Board:  store.NewBoard(ws.Layout),
		Ledger: store.NewLedger(ws.Layout),
		Bus:    bus.New(ws.Layout),
		OnCall: oncall.Service{L: ws.Layout},
	}
}
