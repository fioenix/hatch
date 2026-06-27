package model

// Ledger action enum (spec/ledger.schema.md).
const (
	ActClaim    = "claim"
	ActStart    = "start"
	ActProgress = "progress"
	ActHandoff  = "handoff"
	ActReview   = "review"
	ActDone     = "done"
	ActBlock    = "block"
	ActUnblock  = "unblock"
	ActRevoke   = "revoke"
	ActNote     = "note"
	ActGate     = "gate"
	ActEscalate = "escalate"
)

// Entry is a single append-only ledger record answering who/what/when/why/where.
type Entry struct {
	TS      string // ISO-8601 with offset
	Agent   string // WHO ("human:<name>" allowed)
	Ticket  string // WHERE ("-" for system events)
	Action  string // WHAT
	From    string // lane change "a/ → b/"
	Why     string // WHY (required)
	Result  string // review/gate result
	ToRole  string // handoff target role
	Handoff string // handoff context (required when Action=handoff)
	Branch  string
	Note    string
	CostUSD float64 // cost of this run/action (tracked)
	Tokens  int     // tokens consumed
}
