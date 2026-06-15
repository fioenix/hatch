package wf

import "github.com/fioenix/overclaud/hatch/internal/model"

// Ports: the workflow engine (an application/use-case service) depends on these
// interfaces, not on concrete infrastructure. The filesystem store implements
// them; tests or alternative backends can provide their own. This is the
// dependency-inversion boundary that keeps the engine free of IO concerns.

// Board is the ticket-store port the engine needs.
type Board interface {
	Find(id string, lanes []string) (model.Ticket, bool, error)
	Path(t model.Ticket) string
	Write(t model.Ticket) (string, error)
}

// Ledger is the audit-log port the engine needs.
type Ledger interface {
	Append(e model.Entry) error
	Recent(days int) ([]model.Entry, error)
}
