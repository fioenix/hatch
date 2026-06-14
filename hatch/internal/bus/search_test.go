package bus

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

func TestSearchFiltersAndScope(t *testing.T) {
	b := New(paths.At(t.TempDir()))
	b.Post(Message{Channel: "#design", From: "codex", To: []string{"#design"}, Body: "dùng CSV streaming cho export"})
	b.Post(Message{Channel: "#design", From: "claude-code", To: []string{"#design"}, Body: "đồng ý streaming"})
	b.Post(Message{Channel: "#random", From: "gemini", To: []string{"#random"}, Body: "streaming nhạc trưa nay"})

	// query across all channels
	all, _ := b.Search(SearchOpts{Query: "streaming"})
	if len(all) != 3 {
		t.Fatalf("want 3 streaming hits, got %d", len(all))
	}
	// restrict to a channel
	d, _ := b.Search(SearchOpts{Query: "streaming", Channel: "#design"})
	if len(d) != 2 {
		t.Fatalf("want 2 in #design, got %d", len(d))
	}
	// scope by subscriptions
	b.Subscribe("#design", "codex")
	subs := b.Subscriptions("codex")
	scoped, _ := b.Search(SearchOpts{Query: "streaming", Channels: subs})
	if len(scoped) != 2 {
		t.Fatalf("subscription scope should yield 2, got %d", len(scoped))
	}
	// newest-first + limit
	lim, _ := b.Search(SearchOpts{Query: "streaming", Limit: 1})
	if len(lim) != 1 {
		t.Fatalf("limit should cap to 1, got %d", len(lim))
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	b := New(paths.At(t.TempDir()))
	b.Subscribe("#design", "codex")
	b.Subscribe("#design", "claude-code")
	if got := b.Members("#design"); len(got) != 2 {
		t.Fatalf("want 2 members, got %v", got)
	}
	b.Unsubscribe("#design", "codex")
	if got := b.Members("#design"); len(got) != 1 || got[0] != "claude-code" {
		t.Fatalf("unsubscribe failed: %v", got)
	}
	if subs := b.Subscriptions("claude-code"); len(subs) != 1 || subs[0] != "#design" {
		t.Fatalf("subscriptions wrong: %v", subs)
	}
}
