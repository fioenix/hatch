// Package store implements the filesystem-as-database: reading and writing
// tickets, ledger entries and KB notes under a .hatch/ workspace.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/mdfront"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Board reads and writes tickets within the board directory.
type Board struct{ L paths.Layout }

// NewBoard returns a board bound to a workspace layout.
func NewBoard(l paths.Layout) *Board { return &Board{L: l} }

// readTicket parses a single ticket file and tags it with its lane.
func readTicket(path, lane string) (model.Ticket, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return model.Ticket{}, err
	}
	var t model.Ticket
	body, err := mdfront.Decode(raw, &t)
	if err != nil {
		return model.Ticket{}, fmt.Errorf("%s: %w", path, err)
	}
	t.Body = body
	t.Lane = lane
	return t, nil
}

// ListLane returns tickets in a single lane, sorted by id.
func (b *Board) ListLane(lane string) ([]model.Ticket, error) {
	dir := b.L.Lane(lane)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []model.Ticket
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		t, err := readTicket(filepath.Join(dir, e.Name()), lane)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// List returns all tickets across the given lanes, sorted by id.
func (b *Board) List(lanes []string) ([]model.Ticket, error) {
	var all []model.Ticket
	for _, lane := range lanes {
		ts, err := b.ListLane(lane)
		if err != nil {
			return nil, err
		}
		all = append(all, ts...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all, nil
}

// Find locates a ticket by id across lanes, returning its current lane.
func (b *Board) Find(id string, lanes []string) (model.Ticket, bool, error) {
	for _, lane := range lanes {
		ts, err := b.ListLane(lane)
		if err != nil {
			return model.Ticket{}, false, err
		}
		for _, t := range ts {
			if t.ID == id {
				return t, true, nil
			}
		}
	}
	return model.Ticket{}, false, nil
}

// Path returns the on-disk path a ticket occupies given its current lane.
func (b *Board) Path(t model.Ticket) string {
	return filepath.Join(b.L.Lane(t.Lane), t.Filename())
}

// Write serializes a ticket into its lane directory.
func (b *Board) Write(t model.Ticket) (string, error) {
	dir := b.L.Lane(t.Lane)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	out, err := mdfront.Encode(t, t.Body)
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, t.Filename())
	if err := os.WriteFile(p, out, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// NextID returns the next sequential ticket id (T-NNN) across all lanes.
func (b *Board) NextID(lanes []string) (string, error) {
	max := 0
	for _, lane := range lanes {
		ts, err := b.ListLane(lane)
		if err != nil {
			return "", err
		}
		for _, t := range ts {
			var n int
			if _, err := fmt.Sscanf(t.ID, "T-%d", &n); err == nil && n > max {
				max = n
			}
		}
	}
	return fmt.Sprintf("T-%03d", max+1), nil
}
