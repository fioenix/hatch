// Package roster persists the workspace room's membership: who has joined, the
// session that holds their memory, and whether they are reachable. It is the
// "team simulation" presence layer — distinct from the legacy presence package
// (availability-for-assignment) and from the registry (static config). The
// filesystem is the database: one JSON file per workspace.
package roster

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// IdleAfter is how long since LastSeen before an online member is shown as
// idle. It is cosmetic (idle members are still reachable/wakeable); it just
// tells the boss who is actively around.
const IdleAfter = 5 * time.Minute

// Store reads and writes the roster for one workspace.
type Store struct{ L paths.Layout }

// New returns a roster store bound to a workspace layout.
func New(l paths.Layout) *Store { return &Store{L: l} }

// Load reads the roster, returning an empty one if absent.
func (s *Store) Load() (model.Roster, error) {
	raw, err := os.ReadFile(s.L.Roster())
	if err != nil {
		if os.IsNotExist(err) {
			return model.Roster{}, nil
		}
		return nil, err
	}
	r := model.Roster{}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return r, nil
}

// Save writes the roster atomically-ish (write then rename) to avoid a torn
// read by a concurrently-tailing daemon.
func (s *Store) Save(r model.Roster) error {
	if err := os.MkdirAll(filepath.Dir(s.L.Roster()), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.L.Roster() + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.L.Roster())
}

// Join adds or updates a member, marking it online with a fresh LastSeen. A
// non-empty sessionID records the resumable session that holds the member's
// memory (used by the wake daemon to resume the same teammate). An empty
// sessionID preserves any previously recorded one.
func (s *Store) Join(m model.Member) (model.Member, error) {
	r, err := s.Load()
	if err != nil {
		return model.Member{}, err
	}
	prev := r[m.ID]
	if m.SessionID == "" {
		m.SessionID = prev.SessionID
	}
	if m.Status == "" {
		m.Status = model.MemberOnline
	}
	m.LastSeen = now()
	r[m.ID] = m
	return m, s.Save(r)
}

// Touch refreshes a member's LastSeen and revives it to online if it was idle
// or suspended. Called on any activity (e.g. an MCP tool call) so presence
// reflects reality. A missing or offline member is left unchanged.
func (s *Store) Touch(id string) error {
	r, err := s.Load()
	if err != nil {
		return err
	}
	m, ok := r[id]
	if !ok || m.Status == model.MemberOffline {
		return nil
	}
	m.LastSeen = now()
	if m.Status == model.MemberIdle || m.Status == model.MemberSuspended {
		m.Status = model.MemberOnline
	}
	r[id] = m
	return s.Save(r)
}

// Leave marks a member offline (it has explicitly left the room).
func (s *Store) Leave(id string) error {
	r, err := s.Load()
	if err != nil {
		return err
	}
	if m, ok := r[id]; ok {
		m.Status = model.MemberOffline
		m.LastSeen = now()
		r[id] = m
		return s.Save(r)
	}
	return nil
}

// SetStatus sets a member's status explicitly (e.g. the daemon marking a member
// suspended after it finishes a turn). No-op for a missing member.
func (s *Store) SetStatus(id, status string) error {
	r, err := s.Load()
	if err != nil {
		return err
	}
	if m, ok := r[id]; ok {
		m.Status = status
		m.LastSeen = now()
		r[id] = m
		return s.Save(r)
	}
	return nil
}

// Effective returns the roster with cosmetic idle derivation applied: an online
// member whose LastSeen is older than IdleAfter is shown as idle. It does not
// persist the change. offline/suspended members are left as-is.
func (s *Store) Effective(at time.Time) (model.Roster, error) {
	r, err := s.Load()
	if err != nil {
		return nil, err
	}
	for id, m := range r {
		if m.Status == model.MemberOnline && stale(m.LastSeen, at) {
			m.Status = model.MemberIdle
			r[id] = m
		}
	}
	return r, nil
}

// Members returns the roster's members sorted by id (deterministic listing).
func Members(r model.Roster) []model.Member {
	out := make([]model.Member, 0, len(r))
	for _, m := range r {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func now() string { return time.Now().Format(time.RFC3339) }

func stale(lastSeen string, at time.Time) bool {
	t, err := time.Parse(time.RFC3339, lastSeen)
	if err != nil {
		return false
	}
	return at.Sub(t) > IdleAfter
}
