package model

// Member is one participant present in a workspace's shared room (the roster).
// It is the "team simulation" presence entity: who has joined, what they can
// do, and whether they are reachable right now. It is distinct from a registry
// Agent (static config) — a Member is the live, in-room projection of one.
type Member struct {
	ID        string   `json:"id"`                   // "codex"
	Kind      string   `json:"kind"`                 // claude | codex | agy | kiro | mock | user
	Roles     []string `json:"roles,omitempty"`      // role ids held in the room
	SessionID string   `json:"session_id,omitempty"` // resumable session id (the member's memory); "" if none yet
	Status    string   `json:"status"`               // see member status constants
	LastSeen  string   `json:"last_seen"`            // RFC3339; refreshed on activity
	Note      string   `json:"note,omitempty"`
}

// Member status values. A teammate sleeps (suspended) when idle and is woken on
// a message addressed to it; online/idle distinguish a live seat from a quiet
// one. offline = explicitly left.
const (
	MemberOnline    = "online"
	MemberIdle      = "idle"
	MemberSuspended = "suspended"
	MemberOffline   = "offline"
)

// KindUser marks the human boss as a first-class room member. The boss is never
// a software-driven worker: it is never woken and never counts toward cascade
// depth — a message from the boss starts a fresh episode.
const KindUser = "user"

// Roster is the set of members in a workspace room, keyed by member id.
type Roster map[string]Member

// IsHuman reports whether an id belongs to a human member (the boss). Unknown
// ids are treated as non-human so external/automation actors still gate.
func (r Roster) IsHuman(id string) bool {
	m, ok := r[id]
	return ok && m.Kind == KindUser
}

// Reachable reports whether a member can be woken (present and not offline).
// A missing member is not reachable: it has not joined the room.
func (r Roster) Reachable(id string) bool {
	m, ok := r[id]
	if !ok {
		return false
	}
	return m.Kind != KindUser && m.Status != MemberOffline
}

// WithRole returns the ids of reachable members holding the given role.
func (r Roster) WithRole(role string) []string {
	var out []string
	for id, m := range r {
		if !r.Reachable(id) {
			continue
		}
		for _, rl := range m.Roles {
			if rl == role {
				out = append(out, id)
				break
			}
		}
	}
	return out
}
