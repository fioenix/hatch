// Package oncall tracks the on-call rotation — who is the first responder for
// incidents/escalations right now, and how the duty rotates.
package oncall

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Rotation is the on-call schedule: an ordered list of agents and the index of
// whoever currently holds the pager.
type Rotation struct {
	Order   []string `json:"order"`
	Current int      `json:"current"`
}

// Load reads oncall.json, returning an empty rotation if absent.
func Load(l paths.Layout) Rotation {
	raw, err := os.ReadFile(l.Oncall())
	if err != nil {
		return Rotation{}
	}
	var r Rotation
	if json.Unmarshal(raw, &r) != nil {
		return Rotation{}
	}
	return r
}

// Save persists the rotation.
func (r Rotation) Save(l paths.Layout) error {
	if err := os.MkdirAll(filepath.Dir(l.Oncall()), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Oncall(), append(raw, '\n'), 0o644)
}

// Current returns the agent currently on call, or "" if no rotation is set.
func (r Rotation) Now() string {
	if len(r.Order) == 0 {
		return ""
	}
	return r.Order[r.Current%len(r.Order)]
}

// Rotate advances the pager to the next agent and returns them.
func (r *Rotation) Rotate() string {
	if len(r.Order) == 0 {
		return ""
	}
	r.Current = (r.Current + 1) % len(r.Order)
	return r.Now()
}
