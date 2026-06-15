//go:build hatch_legacy

// Package presence tracks per-agent availability (like Slack presence + PTO):
// who is free to take work, who is busy/paused/offline. Capacity-aware
// assignment uses this plus WIP to route work to a free teammate.
package presence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Availability states.
const (
	Available = "available"
	Busy      = "busy"
	Paused    = "paused"
	Offline   = "offline"
)

// State is one agent's presence.
type State struct {
	Status string `json:"status"`
	Since  string `json:"since"`
	Note   string `json:"note,omitempty"`
}

// Board maps agent id → presence. A missing agent is treated as Available.
type Board map[string]State

// Load reads presence.json, returning an empty board if absent.
func Load(l paths.Layout) Board {
	raw, err := os.ReadFile(l.Presence())
	if err != nil {
		return Board{}
	}
	b := Board{}
	if json.Unmarshal(raw, &b) != nil {
		return Board{}
	}
	return b
}

// Save writes the board to presence.json.
func (b Board) Save(l paths.Layout) error {
	if err := os.MkdirAll(filepath.Dir(l.Presence()), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Presence(), append(raw, '\n'), 0o644)
}

// Set updates an agent's status (and optional note) to now.
func (b Board) Set(agent, status, note string) {
	b[agent] = State{Status: status, Since: time.Now().Format(time.RFC3339), Note: note}
}

// StatusOf returns an agent's status, defaulting to Available.
func (b Board) StatusOf(agent string) string {
	s, ok := b[agent]
	if !ok || s.Status == "" {
		return Available
	}
	return s.Status
}

// CanTakeWork reports whether an agent is free to be assigned (available/busy
// still count as reachable; paused/offline do not).
func (b Board) CanTakeWork(agent string) bool {
	switch b.StatusOf(agent) {
	case Paused, Offline:
		return false
	default:
		return true
	}
}

// Agents returns the agent ids present in the board, sorted.
func (b Board) Agents() []string {
	out := make([]string, 0, len(b))
	for a := range b {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
