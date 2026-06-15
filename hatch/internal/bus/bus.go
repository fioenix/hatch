// Package bus is the agent communication layer: a file-based, append-only,
// auditable message bus that lets agents talk to each other directly — direct
// messages, @mentions, questions, and multi-agent meetings. The bus carries
// *dialogue*; the board/ledger remain the source of truth for *state*. The
// orchestrator is the medium that delivers turns between process-isolated
// agents (like a team's chat server), so every exchange stays on the record.
package bus

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Message types (aliases of the domain constants).
const (
	TypeMsg      = model.MsgText
	TypeAsk      = model.MsgAsk
	TypeReply    = model.MsgReply
	TypeDecision = model.MsgDecision
)

// Message and SearchOpts are the communication domain types; bus re-exports
// them as aliases so call sites stay terse while the canonical types live in
// the domain (model), letting ports return them without coupling to this
// package.
type Message = model.Message

// Bus reads and writes conversation threads under .hatch/bus/.
type Bus struct{ L paths.Layout }

// New returns a bus bound to a workspace layout.
func New(l paths.Layout) *Bus { return &Bus{L: l} }

func (b *Bus) dir() string         { return filepath.Join(b.L.Root, "bus") }
func (b *Bus) threadsDir() string  { return filepath.Join(b.dir(), "threads") }
func (b *Bus) cursorsPath() string { return filepath.Join(b.dir(), ".cursors.json") }

// safeThread maps a thread/channel id to a filename (drops a leading '#').
func safeThread(id string) string {
	id = strings.TrimPrefix(id, "#")
	var s strings.Builder
	for _, r := range strings.ToLower(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			s.WriteRune(r)
		default:
			s.WriteRune('-')
		}
	}
	return strings.Trim(s.String(), "-")
}

func (b *Bus) threadPath(thread string) string {
	return filepath.Join(b.threadsDir(), safeThread(thread)+".md")
}

// render formats a message as an append-only Markdown block.
func render(m Message) string {
	var s strings.Builder
	to := strings.Join(m.To, ", ")
	fmt.Fprintf(&s, "## %s · %s → %s · %s", m.TS, m.From, to, m.Type)
	if m.InReplyTo != "" {
		fmt.Fprintf(&s, " · re:%s", m.InReplyTo)
	}
	fmt.Fprintf(&s, " · {#%s}\n", m.ID)
	s.WriteString(strings.TrimRight(m.Body, "\n"))
	s.WriteString("\n")
	return s.String()
}

// Post appends a message to its thread, defaulting ID and timestamp.
func (b *Bus) Post(m Message) (Message, error) {
	if m.Channel == "" {
		return m, fmt.Errorf("message requires a channel")
	}
	if m.From == "" {
		return m, fmt.Errorf("message requires a sender")
	}
	if strings.TrimSpace(m.Body) == "" {
		return m, fmt.Errorf("message body is empty")
	}
	if m.Type == "" {
		m.Type = TypeMsg
	}
	// @mentions in the body tag teammates (agent ids or roles) just like Slack.
	for _, tag := range Mentions(m.Body) {
		if !contains(m.To, tag) {
			m.To = append(m.To, tag)
		}
	}
	if m.TS == "" {
		m.TS = time.Now().Format(time.RFC3339Nano)
	}
	if m.ID == "" {
		m.ID = newID(m.TS)
	}
	if err := os.MkdirAll(b.threadsDir(), 0o755); err != nil {
		return m, err
	}
	p := b.threadPath(m.Channel)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return m, err
	}
	defer f.Close()
	block := render(m)
	if fi, _ := f.Stat(); fi != nil && fi.Size() > 0 {
		block = "\n" + block
	}
	_, err = f.WriteString(block)
	return m, err
}

// Messages returns the parsed messages of a channel in order.
func (b *Bus) Messages(channel string) ([]Message, error) {
	raw, err := os.ReadFile(b.threadPath(channel))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseThread(safeThread(channel), string(raw)), nil
}

// Raw returns the raw markdown of a channel (for handing to an agent as context).
func (b *Bus) Raw(channel string) (string, error) {
	raw, err := os.ReadFile(b.threadPath(channel))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(raw), nil
}

// Channels lists all channel/conversation ids.
func (b *Bus) Channels() ([]string, error) {
	ents, err := os.ReadDir(b.threadsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Replies returns a threaded sub-conversation within a channel: the root
// message plus every message replying to it (one level).
func (b *Bus) Replies(channel, rootID string) ([]Message, error) {
	msgs, err := b.Messages(channel)
	if err != nil {
		return nil, err
	}
	var out []Message
	for _, m := range msgs {
		if m.ID == rootID || m.InReplyTo == rootID {
			out = append(out, m)
		}
	}
	return out, nil
}

// Inbox returns messages addressed to an agent (by id, one of its roles, "*",
// or "all") across all threads, newer than the agent's cursor.
func (b *Bus) Inbox(agent string, roles []string) ([]Message, error) {
	cursors, _ := b.loadCursors()
	var sinceT time.Time
	if s := cursors[agent]; s != "" {
		sinceT, _ = time.Parse(time.RFC3339Nano, s)
	}
	want := map[string]bool{agent: true, "*": true, "all": true}
	for _, r := range roles {
		want[r] = true
	}
	channels, err := b.Channels()
	if err != nil {
		return nil, err
	}
	var out []Message
	for _, ch := range channels {
		msgs, err := b.Messages(ch)
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			if m.From == agent {
				continue
			}
			if !sinceT.IsZero() {
				if mt, err := time.Parse(time.RFC3339Nano, m.TS); err == nil && !mt.After(sinceT) {
					continue
				}
			}
			for _, to := range m.To {
				if want[to] {
					out = append(out, m)
					break
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out, nil
}

// MarkRead advances an agent's cursor to now.
func (b *Bus) MarkRead(agent string) error {
	cursors, _ := b.loadCursors()
	cursors[agent] = time.Now().Format(time.RFC3339Nano)
	return b.saveCursors(cursors)
}

func newID(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t = time.Now()
	}
	// microsecond suffix keeps ids unique within the same second.
	return fmt.Sprintf("m%s-%06d", t.Format("0102-150405"), t.Nanosecond()/1000)
}
