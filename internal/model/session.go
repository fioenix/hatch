package model

// Session is one agent's resumable CLI conversation about one bus thread — the
// "warm cache" of that teammate's work on that task. The shared source of truth
// is the bus + KB; a Session just lets an agent resume its own context instead
// of re-reading the record every wake. Agents whose CLI cannot expose a session
// id (e.g. agy) run stateless and have no Session.
type Session struct {
	Agent         string   `json:"agent"`             // member id, e.g. "codex"
	Thread        string   `json:"thread"`            // bus channel id this session serves
	Kind          string   `json:"kind"`              // claude | codex (kinds that keep warm sessions)
	ID            string   `json:"id"`                // CLI session id (UUID); "" until known
	Status        string   `json:"status"`            // see session status constants
	StartedAt     string   `json:"started_at"`        // RFC3339
	LastResumedAt string   `json:"last_resumed_at"`   // RFC3339
	History       []string `json:"history,omitempty"` // superseded ids, newest last (audit)
}

// Session status values. A live session is resumable; a stale one failed to
// resume and will be replaced by a fresh session on the next wake.
const (
	SessionLive  = "live"
	SessionStale = "stale"
	SessionEnded = "ended"
)
