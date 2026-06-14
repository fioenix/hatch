package model

// Priority levels for a ticket.
const (
	P0 = "P0"
	P1 = "P1"
	P2 = "P2"
	P3 = "P3"
)

// Claim is the lightweight lock placed on a ticket when an agent picks it up.
type Claim struct {
	Agent string `yaml:"agent"`
	TS    string `yaml:"ts"`
}

// ExternalBlocker is a dependency on something outside the squad (a vendor,
// another team, a human approval) — surfaced as risk, not auto-resolved.
type ExternalBlocker struct {
	What   string `yaml:"what"`
	Owner  string `yaml:"owner,omitempty"`
	ETA    string `yaml:"eta,omitempty"`
	Status string `yaml:"status,omitempty"` // waiting | received
}

// Ticket mirrors spec/ticket.schema.md. The directory it lives in (the lane)
// is the source of truth for its lifecycle state; Status must agree with it.
type Ticket struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Status      string   `yaml:"status"`
	Role        string   `yaml:"role"`
	Assignee    string   `yaml:"assignee,omitempty"`
	Priority    string   `yaml:"priority,omitempty"`
	Epic        string   `yaml:"epic,omitempty"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	Branch      string   `yaml:"branch,omitempty"`
	ContextRefs []string `yaml:"context_refs,omitempty"`
	Claim       *Claim   `yaml:"claim,omitempty"`
	DoD         []string `yaml:"dod,omitempty"`

	BlockedByExternal []ExternalBlocker `yaml:"blocked_by_external,omitempty"`
	Created           string            `yaml:"created,omitempty"`
	Updated           string            `yaml:"updated,omitempty"`

	// Body and Lane are populated by the store, not the frontmatter.
	Body string `yaml:"-"`
	Lane string `yaml:"-"`
}

// Filename is the canonical file name for a ticket on the board.
func (t Ticket) Filename() string { return t.ID + ".md" }

// OpenExternal returns external blockers not yet received.
func (t Ticket) OpenExternal() []ExternalBlocker {
	var out []ExternalBlocker
	for _, e := range t.BlockedByExternal {
		if e.Status != "received" {
			out = append(out, e)
		}
	}
	return out
}
