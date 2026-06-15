//go:build hatch_legacy

package wf

import "github.com/fioenix/overclaud/hatch/internal/port"

// Engine is the workflow use-case service. It depends only on ports, never on
// concrete infrastructure — the composition root injects adapters. This keeps
// the engine free of IO concerns (see ARCHITECTURE.md).
type Engine struct {
	Board  port.Board
	Ledger port.Ledger
	Bus    port.Bus
	OnCall port.OnCall
}
