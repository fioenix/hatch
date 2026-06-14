package bus

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

func newBus(t *testing.T) *Bus {
	t.Helper()
	return New(paths.At(t.TempDir()))
}

func TestPostAndParseRoundTrip(t *testing.T) {
	b := newBus(t)
	if _, err := b.Post(Message{Channel: "T-1", From: "codex", To: []string{"claude-code", "reviewer"}, Type: TypeAsk, Body: "Có nên dùng streaming?\nDòng 2."}); err != nil {
		t.Fatal(err)
	}
	msgs, err := b.Messages("T-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(msgs))
	}
	m := msgs[0]
	if m.From != "codex" || m.Type != TypeAsk {
		t.Fatalf("parsed wrong: %+v", m)
	}
	if len(m.To) != 2 || m.To[0] != "claude-code" || m.To[1] != "reviewer" {
		t.Fatalf("recipients wrong: %v", m.To)
	}
	if m.Body != "Có nên dùng streaming?\nDòng 2." {
		t.Fatalf("body wrong: %q", m.Body)
	}
	if m.ID == "" {
		t.Fatal("id not assigned/parsed")
	}
}

func TestInboxMatchesIdRoleAndStar(t *testing.T) {
	b := newBus(t)
	b.Post(Message{Channel: "t", From: "a", To: []string{"reviewer"}, Body: "by role"})
	b.Post(Message{Channel: "t", From: "a", To: []string{"claude-code"}, Body: "by id"})
	b.Post(Message{Channel: "t", From: "a", To: []string{"*"}, Body: "broadcast"})
	b.Post(Message{Channel: "t", From: "a", To: []string{"codex"}, Body: "not for me"})

	in, err := b.Inbox("claude-code", []string{"reviewer", "architect"})
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 3 {
		t.Fatalf("want 3 inbox msgs (role+id+star), got %d", len(in))
	}
}

func TestInboxExcludesOwnAndRespectsCursor(t *testing.T) {
	b := newBus(t)
	b.Post(Message{Channel: "t", From: "claude-code", To: []string{"*"}, Body: "my own"})
	b.Post(Message{Channel: "t", From: "codex", To: []string{"claude-code"}, Body: "first"})
	if err := b.MarkRead("claude-code"); err != nil {
		t.Fatal(err)
	}
	b.Post(Message{Channel: "t", From: "codex", To: []string{"claude-code"}, Body: "after read"})

	in, err := b.Inbox("claude-code", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 1 || in[0].Body != "after read" {
		t.Fatalf("cursor/own-exclusion wrong: %+v", in)
	}
}

func TestMentionsRouteToInbox(t *testing.T) {
	b := newBus(t)
	// no explicit --to; tagging via @mention in the body.
	b.Post(Message{Channel: "#design", From: "codex", To: []string{"#design"},
		Body: "@claude-code @tester nên dùng streaming nhé"})
	in, err := b.Inbox("claude-code", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 1 {
		t.Fatalf("mention @claude-code should hit inbox, got %d", len(in))
	}
	// role mention reaches an agent holding that role.
	tin, _ := b.Inbox("codexer", []string{"tester"})
	if len(tin) != 1 {
		t.Fatalf("@tester role mention should reach a tester, got %d", len(tin))
	}
}

func TestMentionsExtract(t *testing.T) {
	got := Mentions("hey @codex and @claude-code, ask @reviewer; email a@b.com not a mention start")
	want := map[string]bool{"codex": true, "claude-code": true, "reviewer": true}
	for _, g := range got {
		delete(want, g)
	}
	if len(want) != 0 {
		t.Fatalf("missing mentions: %v (got %v)", want, got)
	}
}

func TestBodyWithMarkdownHeadingNotSplit(t *testing.T) {
	b := newBus(t)
	body := "Đề xuất:\n## Phương án A\nchi tiết\n## Phương án B\nkhác"
	b.Post(Message{Channel: "#design", From: "codex", To: []string{"*"}, Body: body})
	msgs, err := b.Messages("#design")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("body with ## headings must stay 1 message, got %d", len(msgs))
	}
	if !strings.Contains(msgs[0].Body, "Phương án B") {
		t.Fatalf("body truncated: %q", msgs[0].Body)
	}
}
