package slack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/fioenix/hatch/internal/paths"
)

// threadmap remembers which Slack thread (thread_ts) carries each bus channel,
// so a channel's messages collapse into one Slack thread (1 task = 1 thread),
// plus the mirror cursor (last bus TS mirrored) so a bridge restart does not
// re-post history. It is shared by the inbound (Socket Mode) and outbound
// (ticker) goroutines, so every access is guarded by mu.
type threadmap struct {
	path   string
	mu     sync.Mutex
	fwd    map[string]string // bus channel → slack thread_ts
	rev    map[string]string // slack thread_ts → bus channel
	cursor string            // RFC3339Nano of the last bus message mirrored out
}

// persisted is the on-disk shape of a threadmap.
type persisted struct {
	Cursor  string            `json:"cursor,omitempty"`
	Threads map[string]string `json:"threads"`
}

func loadThreadmap(l paths.Layout) *threadmap {
	tm := &threadmap{path: l.SlackThreadmap(), fwd: map[string]string{}, rev: map[string]string{}}
	if raw, err := os.ReadFile(tm.path); err == nil {
		var p persisted
		if json.Unmarshal(raw, &p) == nil {
			tm.cursor = p.Cursor
			for ch, ts := range p.Threads {
				tm.fwd[ch] = ts
				tm.rev[ts] = ch
			}
		}
	}
	return tm
}

// newMemThreadmap returns a throwaway, in-memory map (no persistence) — used by
// dry-run so it never touches the real threadmap.json.
func newMemThreadmap() *threadmap {
	return &threadmap{fwd: map[string]string{}, rev: map[string]string{}}
}

func (t *threadmap) tsFor(channel string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ts, ok := t.fwd[channel]
	return ts, ok
}

func (t *threadmap) channelFor(ts string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ch, ok := t.rev[ts]
	return ch, ok
}

func (t *threadmap) getCursor() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cursor
}

// bind records channel↔ts and persists.
func (t *threadmap) bind(channel, ts string) error {
	if ts == "" {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fwd[channel] = ts
	t.rev[ts] = channel
	return t.save()
}

// setCursor records the mirror high-water mark and persists.
func (t *threadmap) setCursor(ts string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cursor = ts
	return t.save()
}

// save writes the whole map atomically. Caller holds mu. In-memory maps
// (path == "", dry-run) never touch disk.
func (t *threadmap) save() error {
	if t.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(persisted{Cursor: t.cursor, Threads: t.fwd}, "", "  ")
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
