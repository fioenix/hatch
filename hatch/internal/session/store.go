// Package session manages agents' resumable CLI sessions, keyed by (agent,
// bus-thread). It is the lifecycle layer the wake daemon uses to resume a
// teammate's warm context for a specific task. The bus + KB remain the source
// of truth; a session is only a cache pointer, so losing one degrades to a
// fresh read of the record, never to lost work.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// book is the on-disk shape: agent id → bus thread → session.
type book map[string]map[string]model.Session

// mu serializes read-modify-write of sessions.json within this process (the
// daemon resumes several agents concurrently). Reads are lock-free (writes end
// with an atomic rename).
var mu sync.Mutex

// Store reads and writes .hatch/sessions.json.
type Store struct{ L paths.Layout }

// New returns a store bound to a workspace layout.
func New(l paths.Layout) *Store { return &Store{L: l} }

func (s *Store) load() (book, error) {
	raw, err := os.ReadFile(s.L.Sessions())
	if err != nil {
		if os.IsNotExist(err) {
			return book{}, nil
		}
		return nil, err
	}
	var b book
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	if b == nil {
		b = book{}
	}
	return b, nil
}

func (s *Store) save(b book) error {
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.L.Sessions()), 0o755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", s.L.Sessions(), os.Getpid())
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.L.Sessions())
}

// Get returns the session for (agent, thread), if any.
func (s *Store) Get(agent, thread string) (model.Session, bool) {
	b, err := s.load()
	if err != nil {
		return model.Session{}, false
	}
	sess, ok := b[agent][thread]
	return sess, ok
}

// Put upserts a session by (Agent, Thread). When it supersedes a session with a
// different id, the old id is pushed onto History for audit.
func (s *Store) Put(sess model.Session) error {
	mu.Lock()
	defer mu.Unlock()
	b, err := s.load()
	if err != nil {
		return err
	}
	if b[sess.Agent] == nil {
		b[sess.Agent] = map[string]model.Session{}
	}
	if prev, ok := b[sess.Agent][sess.Thread]; ok && prev.ID != "" && prev.ID != sess.ID {
		sess.History = append(append([]string{}, prev.History...), prev.ID)
	}
	b[sess.Agent][sess.Thread] = sess
	return s.save(b)
}

// MarkStale flags a session unresumable so the next wake starts fresh.
func (s *Store) MarkStale(agent, thread string) error {
	mu.Lock()
	defer mu.Unlock()
	b, err := s.load()
	if err != nil {
		return err
	}
	sess, ok := b[agent][thread]
	if !ok {
		return nil
	}
	sess.Status = model.SessionStale
	b[agent][thread] = sess
	return s.save(b)
}

// All returns every recorded session, sorted by agent then thread.
func (s *Store) All() []model.Session {
	b, err := s.load()
	if err != nil {
		return nil
	}
	var out []model.Session
	for _, threads := range b {
		for _, sess := range threads {
			out = append(out, sess)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Agent != out[j].Agent {
			return out[i].Agent < out[j].Agent
		}
		return out[i].Thread < out[j].Thread
	})
	return out
}

// Now returns an RFC3339 timestamp (exported so callers stamp consistently).
func Now() string { return time.Now().Format(time.RFC3339) }
