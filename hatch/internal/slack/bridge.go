package slack

import (
	"sort"
	"strings"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
	"github.com/fioenix/overclaud/hatch/internal/roster"
)

// poster sends one message to the configured Slack channel. threadTS is "" for
// a new thread root, or the thread_ts to reply under. It returns the new
// message's ts. The concrete impl wraps slack.Client; tests use a fake.
type poster interface {
	post(threadTS, username, icon, text string) (ts string, err error)
}

// incoming is a normalised Slack message event, decoupled from slack-go so the
// IN path is a pure, testable method.
type incoming struct {
	ChannelID string
	User      string
	BotID     string
	SubType   string
	ThreadTS  string
	TS        string
	Text      string
}

// Bridge mirrors the bus into Slack (OUT) and ingests the boss's Slack messages
// onto the bus (IN). It owns no orchestration: IN just writes a peer message
// that the wake daemon later delivers.
type Bridge struct {
	Bus    *bus.Bus
	Roster *roster.Store
	Cfg    Config

	poster poster
	tm     *threadmap
	cursor time.Time // newest bus TS already mirrored to Slack
}

// NewBridge wires a bridge. tm and p are injected so tests can supply fakes.
func NewBridge(b *bus.Bus, rs *roster.Store, cfg Config, p poster, tm *threadmap) *Bridge {
	return &Bridge{Bus: b, Roster: rs, Cfg: cfg, poster: p, tm: tm}
}

// mirrorOnce posts every bus message newer than the cursor into Slack, skipping
// the boss's own messages (already visible in Slack — and the loop-break for
// IN). Each channel collapses into one Slack thread. On a post error it returns
// with the cursor advanced past whatever already succeeded, so a retry will not
// duplicate.
func (b *Bridge) mirrorOnce(now time.Time) error {
	r, err := b.Roster.Effective(now)
	if err != nil {
		r = model.Roster{} // identity falls back to ids; mirroring still works
	}
	msgs := b.tailBus()
	for _, m := range msgs {
		t := parseTS(m.TS)
		if m.From == b.Cfg.Boss {
			b.advance(t)
			continue
		}
		name, icon := identity(r, m.From)
		text := m.Body
		threadTS, known := b.tm.tsFor(m.Channel)
		if !known {
			text = "*#" + m.Channel + "*\n" + text // header on the thread root
		}
		ts, perr := b.poster.post(threadTS, name, icon, text)
		if perr != nil {
			return perr
		}
		if !known {
			_ = b.tm.bind(m.Channel, ts)
		}
		b.advance(t)
	}
	return nil
}

// handleIncoming turns a genuine human Slack message into a bus post from the
// boss. bus.Post extracts @mentions from the text, so a literal "@codex …"
// reaches the daemon exactly like a peer message. Bot/echo/system events are
// dropped (this is the OUT→IN loop break together with the From==boss skip).
func (b *Bridge) handleIncoming(in incoming) error {
	if in.ChannelID != b.Cfg.ChannelID {
		return nil // not our room
	}
	if in.BotID != "" || in.User == "" || in.SubType != "" {
		return nil // our own posts, other bots, or system/edit events
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil
	}
	ch := b.channelForIncoming(in)
	_, err := b.Bus.Post(model.Message{
		Channel: ch,
		From:    b.Cfg.Boss,
		Type:    bus.TypeMsg,
		Body:    text,
	})
	return err
}

// channelForIncoming resolves which bus channel a Slack message belongs to. A
// thread reply maps back through the threadmap; a top-level message opens a new
// channel and binds its ts so agent replies (OUT) nest under it.
func (b *Bridge) channelForIncoming(in incoming) string {
	if in.ThreadTS != "" {
		if ch, ok := b.tm.channelFor(in.ThreadTS); ok {
			return ch
		}
		ch := "t-" + paths.SafeSegment(in.ThreadTS)
		_ = b.tm.bind(ch, in.ThreadTS)
		return ch
	}
	ch := "t-" + paths.SafeSegment(in.TS)
	_ = b.tm.bind(ch, in.TS)
	return ch
}

func (b *Bridge) advance(t time.Time) {
	if t.After(b.cursor) {
		b.cursor = t
	}
}

// tailBus returns bus messages newer than the cursor, across all channels,
// sorted by TS.
func (b *Bridge) tailBus() []model.Message {
	chans, err := b.Bus.Channels()
	if err != nil {
		return nil
	}
	var out []model.Message
	for _, ch := range chans {
		msgs, err := b.Bus.Messages(ch)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			if parseTS(m.TS).After(b.cursor) {
				out = append(out, m)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out
}

func parseTS(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}
