package slack

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// threadmap remembers which Slack thread (thread_ts) carries each bus channel,
// so a channel's messages collapse into one Slack thread (1 task = 1 thread).
// It is persisted as ch→ts; the reverse (ts→ch) is rebuilt in memory for IN.
type threadmap struct {
	path string
	fwd  map[string]string // bus channel → slack thread_ts
	rev  map[string]string // slack thread_ts → bus channel
}

func loadThreadmap(l paths.Layout) *threadmap {
	tm := &threadmap{path: l.SlackThreadmap(), fwd: map[string]string{}, rev: map[string]string{}}
	if raw, err := os.ReadFile(tm.path); err == nil {
		_ = json.Unmarshal(raw, &tm.fwd)
	}
	for ch, ts := range tm.fwd {
		tm.rev[ts] = ch
	}
	return tm
}

// newMemThreadmap returns a throwaway, in-memory map (no persistence) — used by
// dry-run so it never touches the real threadmap.json.
func newMemThreadmap() *threadmap {
	return &threadmap{fwd: map[string]string{}, rev: map[string]string{}}
}

func (t *threadmap) tsFor(channel string) (string, bool) { ts, ok := t.fwd[channel]; return ts, ok }
func (t *threadmap) channelFor(ts string) (string, bool) { ch, ok := t.rev[ts]; return ch, ok }

// bind records channel↔ts and persists. Atomic write-then-rename keeps the file
// readable even if the process dies mid-write.
func (t *threadmap) bind(channel, ts string) error {
	if ts == "" {
		return nil
	}
	t.fwd[channel] = ts
	t.rev[ts] = channel
	if t.path == "" {
		return nil // in-memory map (dry-run): never persist
	}
	raw, err := json.MarshalIndent(t.fwd, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return err
	}
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, t.path)
}
