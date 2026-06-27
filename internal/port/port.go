// Package port declares the interfaces (ports) the use-case layer depends on.
// Infrastructure packages (store, bus, oncall, …) provide adapters that satisfy
// them; the CLI/TUI composition root wires concrete adapters into use cases.
// Keeping these here makes the dependency-inversion boundary explicit and lets
// the core be tested with fakes. See ARCHITECTURE.md.
package port

import "github.com/fioenix/hatch/internal/model"

// Ledger is the append-only audit-log port.
type Ledger interface {
	Append(e model.Entry) error
	Recent(days int) ([]model.Entry, error)
	Entries() ([]model.Entry, error)
}

// Bus is the communication port the use-case layer needs: post a message,
// build a per-agent catch-up (inbox + query-scoped recall) as formatted lines,
// advance a read cursor, and search the conversation history.
type Bus interface {
	Notify(channel, from string, to []string, body string) error
	CatchUp(agent string, roles []string, query string, limit int) (inbox []string, recall []string)
	MarkRead(agent string) error
	Search(o model.SearchOpts) ([]model.Message, error)
}

// KB is the knowledge-base port the use-case layer reads from.
type KB interface {
	List() ([]model.KBEntry, error)
}

// OnCall reports who currently holds the pager.
type OnCall interface {
	Current() string
}
