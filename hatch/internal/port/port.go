// Package port declares the interfaces (ports) the use-case layer depends on.
// Infrastructure packages (store, bus, oncall, …) provide adapters that satisfy
// them; the CLI/TUI composition root wires concrete adapters into use cases.
// Keeping these here makes the dependency-inversion boundary explicit and lets
// the core be tested with fakes. See ARCHITECTURE.md.
package port

import "github.com/fioenix/overclaud/hatch/internal/model"

// Board is the ticket-store port.
type Board interface {
	ListLane(lane string) ([]model.Ticket, error)
	Find(id string, lanes []string) (model.Ticket, bool, error)
	Path(t model.Ticket) string
	Write(t model.Ticket) (string, error)
}

// Ledger is the append-only audit-log port.
type Ledger interface {
	Append(e model.Entry) error
	Recent(days int) ([]model.Entry, error)
	Entries() ([]model.Entry, error)
}

// Bus is the communication port the use-case layer needs: post a message,
// build a per-agent catch-up (inbox + query-scoped recall) as formatted lines,
// and advance an agent's read cursor.
type Bus interface {
	Notify(channel, from string, to []string, body string) error
	CatchUp(agent string, roles []string, query string, limit int) (inbox []string, recall []string)
	MarkRead(agent string) error
}

// OnCall reports who currently holds the pager.
type OnCall interface {
	Current() string
}
