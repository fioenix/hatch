package slack

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
	"github.com/fioenix/overclaud/hatch/internal/roster"
)

type fakePost struct{ from, threadTS, username, icon, text string }

type fakePoster struct {
	posts []fakePost
	n     int
}

func (f *fakePoster) post(from, threadTS, username, icon, text string) (string, error) {
	f.posts = append(f.posts, fakePost{from, threadTS, username, icon, text})
	f.n++
	return fmt.Sprintf("ts-%d", f.n), nil
}

func newTestBridge(t *testing.T, cfg Config) (*Bridge, *bus.Bus, *roster.Store, *fakePoster) {
	t.Helper()
	return newTestBridgeM(t, cfg, nil)
}

func newTestBridgeM(t *testing.T, cfg Config, mentions map[string]string) (*Bridge, *bus.Bus, *roster.Store, *fakePoster) {
	t.Helper()
	l := paths.At(t.TempDir())
	b := bus.New(l)
	rs := roster.New(l)
	fp := &fakePoster{}
	return NewBridge(b, rs, cfg, fp, loadThreadmap(l), mentions), b, rs, fp
}

// OUT: agent messages mirror (one thread per bus channel), the boss's own
// messages are skipped, and the first message of a channel carries a header.
func TestMirrorOut(t *testing.T) {
	br, b, _, fp := newTestBridge(t, Config{Boss: "fioenix", ChannelID: "C1"})
	mustPost(t, b, "task-1", "codex", "hello")
	mustPost(t, b, "task-1", "claude", "hi back")
	mustPost(t, b, "task-1", "fioenix", "boss talking") // skipped: boss sees own msg in Slack

	if err := br.mirrorOnce(time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(fp.posts) != 2 {
		t.Fatalf("want 2 mirrored posts, got %d: %+v", len(fp.posts), fp.posts)
	}
	if fp.posts[0].threadTS != "" {
		t.Errorf("first post should open a thread (empty threadTS), got %q", fp.posts[0].threadTS)
	}
	if !strings.Contains(fp.posts[0].text, "*#task-1*") {
		t.Errorf("thread root should carry channel header, got %q", fp.posts[0].text)
	}
	if fp.posts[1].threadTS != "ts-1" {
		t.Errorf("second post should reply under ts-1, got %q", fp.posts[1].threadTS)
	}
	if fp.posts[1].from != "claude" {
		t.Errorf("want post attributed to agent claude, got %q", fp.posts[1].from)
	}
}

// IN: a native Slack mention "<@Ucodex>" is rewritten to "@codex" so bus.Post
// routes it; an unmapped mention (a real human) is left untouched.
func TestIngestTranslatesMention(t *testing.T) {
	br, b, _, _ := newTestBridgeM(t,
		Config{Boss: "fioenix", ChannelID: "C1"},
		map[string]string{"UCODEX": "codex"})
	err := br.handleIncoming(incoming{ChannelID: "C1", User: "U1", TS: "1700.5",
		Text: "<@UCODEX> and <@UHUMAN> please look"})
	if err != nil {
		t.Fatal(err)
	}
	msgs, _ := b.Messages("t-1700.5")
	if len(msgs) != 1 {
		t.Fatalf("want 1 bus message, got %d", len(msgs))
	}
	if !contains(msgs[0].To, "codex") {
		t.Errorf("want @codex routed from <@UCODEX>, got To=%v", msgs[0].To)
	}
	if !strings.Contains(msgs[0].Body, "<@UHUMAN>") {
		t.Errorf("unmapped mention should survive, got %q", msgs[0].Body)
	}
}

// A second mirror pass must not re-post messages already seen (cursor advances).
func TestMirrorIdempotent(t *testing.T) {
	br, b, _, fp := newTestBridge(t, Config{Boss: "fioenix", ChannelID: "C1"})
	mustPost(t, b, "task-1", "codex", "hello")
	_ = br.mirrorOnce(time.Now())
	_ = br.mirrorOnce(time.Now())
	if len(fp.posts) != 1 {
		t.Fatalf("want 1 post across two passes, got %d", len(fp.posts))
	}
}

// Impersonation prefers a member's Note as display name.
func TestMirrorIdentityFromRoster(t *testing.T) {
	br, b, rs, fp := newTestBridge(t, Config{Boss: "fioenix", ChannelID: "C1"})
	if _, err := rs.Join(model.Member{ID: "codex", Kind: "codex", Note: "Codex (impl)"}); err != nil {
		t.Fatal(err)
	}
	mustPost(t, b, "task-1", "codex", "hello")
	_ = br.mirrorOnce(time.Now())
	if fp.posts[0].username != "Codex (impl)" {
		t.Errorf("want display name from Note, got %q", fp.posts[0].username)
	}
	if fp.posts[0].icon != ":gear:" {
		t.Errorf("want codex icon, got %q", fp.posts[0].icon)
	}
}

// IN: a top-level Slack message opens a new bus channel from the boss, with the
// @mention parsed so the daemon can route it; the thread is bound so agent
// replies nest under the boss's message.
func TestIngestTopLevel(t *testing.T) {
	br, b, _, _ := newTestBridge(t, Config{Boss: "fioenix", ChannelID: "C1"})
	err := br.handleIncoming(incoming{ChannelID: "C1", User: "U1", TS: "1700.5", Text: "@codex fix the parser"})
	if err != nil {
		t.Fatal(err)
	}
	msgs, _ := b.Messages("t-1700.5")
	if len(msgs) != 1 {
		t.Fatalf("want 1 bus message, got %d", len(msgs))
	}
	if msgs[0].From != "fioenix" {
		t.Errorf("want From=fioenix, got %q", msgs[0].From)
	}
	if !contains(msgs[0].To, "codex") {
		t.Errorf("want @codex parsed into To, got %v", msgs[0].To)
	}
	if ts, ok := br.tm.tsFor("t-1700.5"); !ok || ts != "1700.5" {
		t.Errorf("boss msg ts should be bound as the thread root, got %q ok=%v", ts, ok)
	}
}

// IN: a Slack reply inside a known thread routes to that thread's bus channel.
func TestIngestThreadReply(t *testing.T) {
	br, b, _, _ := newTestBridge(t, Config{Boss: "fioenix", ChannelID: "C1"})
	mustPost(t, b, "task-1", "codex", "hello")
	_ = br.mirrorOnce(time.Now()) // binds task-1 → ts-1
	err := br.handleIncoming(incoming{ChannelID: "C1", User: "U1", ThreadTS: "ts-1", TS: "x", Text: "@codex looks good"})
	if err != nil {
		t.Fatal(err)
	}
	msgs, _ := b.Messages("task-1")
	if len(msgs) != 2 {
		t.Fatalf("boss reply should land in task-1, got %d msgs", len(msgs))
	}
	if msgs[1].From != "fioenix" {
		t.Errorf("want boss reply, got %q", msgs[1].From)
	}
}

// IN: echoes of our own posts (bot_id), other channels, and system subtypes are
// dropped — this is the OUT→IN loop break.
func TestIngestDrops(t *testing.T) {
	br, b, _, _ := newTestBridge(t, Config{Boss: "fioenix", ChannelID: "C1"})
	cases := []incoming{
		{ChannelID: "C1", BotID: "B123", User: "U1", TS: "1", Text: "echo of agent post"},
		{ChannelID: "C-other", User: "U1", TS: "2", Text: "different channel"},
		{ChannelID: "C1", User: "U1", SubType: "channel_join", TS: "3", Text: "joined"},
		{ChannelID: "C1", User: "U1", TS: "4", Text: "   "},
	}
	for _, in := range cases {
		if err := br.handleIncoming(in); err != nil {
			t.Fatalf("handleIncoming(%+v): %v", in, err)
		}
	}
	chans, _ := b.Channels()
	if len(chans) != 0 {
		t.Fatalf("no message should have reached the bus, got channels %v", chans)
	}
}

func mustPost(t *testing.T, b *bus.Bus, channel, from, body string) {
	t.Helper()
	if _, err := b.Post(bus.Message{Channel: channel, From: from, Body: body}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // ensure monotonic RFC3339Nano timestamps
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
