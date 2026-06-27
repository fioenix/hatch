package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fioenix/hatch/internal/config"
	"github.com/fioenix/hatch/internal/paths"
	"github.com/fioenix/hatch/internal/scaffold"
)

// newWorkspace scaffolds a fresh Hatch workspace in a temp dir.
func newWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"}); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	return dir
}

// connect scaffolds a workspace in a temp dir and connects a session as `me`.
func connect(t *testing.T, me string) *mcp.ClientSession {
	t.Helper()
	return connectIn(t, newWorkspace(t), me)
}

// connectIn builds the Hatch MCP server for agent `me` against an existing
// workspace dir and wires an in-memory client session to it. Multiple agents
// can connect to the same dir to exercise cross-agent chat.
func connectIn(t *testing.T, dir, me string) *mcp.ClientSession {
	t.Helper()
	l, err := paths.Find(dir)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	ws, err := config.Load(l)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	srv := New(ws, me, "test")
	ct, st := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %s", name, text(res))
	}
	return res
}

// structured decodes a tool result's JSON content into the typed Out value.
// The SDK populates Content with JSON text mirroring the structured output.
func structured[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	var out T
	if err := json.Unmarshal([]byte(text(res)), &out); err != nil {
		t.Fatalf("decode result %q: %v", text(res), err)
	}
	return out
}

func text(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestToolsRegistered(t *testing.T) {
	cs := connect(t, "claude-code")
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	for _, want := range []string{
		"whoami", "chat_open", "chat_post", "chat_read",
		"chat_inbox", "chat_search", "chat_channels", "kb_add", "kb_search",
	} {
		if !got[want] {
			t.Errorf("missing tool %q", want)
		}
	}
}

func TestChatRoundTrip(t *testing.T) {
	cs := connect(t, "claude-code")

	// Open a thread (= a task) and post a reply into it.
	open := call(t, cs, "chat_open", openIn{Title: "Export CSV", Body: "@codex giúp streaming"})
	ch := structured[postOut](t, open).Channel
	if ch == "" {
		t.Fatal("chat_open returned empty channel")
	}
	if got := structured[postOut](t, open).MessageID; got == "" {
		t.Fatal("chat_open returned empty message id")
	}

	post := call(t, cs, "chat_post", postIn{Channel: ch, Body: "done, PR up"})
	if structured[postOut](t, post).Channel != ch {
		t.Fatalf("chat_post channel mismatch: %s", text(post))
	}

	// chat_read should contain both messages.
	read := call(t, cs, "chat_read", channelIn{Channel: ch})
	body := structured[textOut](t, read).Text
	if !strings.Contains(body, "Export CSV") || !strings.Contains(body, "PR up") {
		t.Fatalf("chat_read missing content: %q", body)
	}

	// The channel shows up in chat_channels.
	chans := call(t, cs, "chat_channels", struct{}{})
	if !contains(structured[channelsOut](t, chans).Channels, ch) {
		t.Fatalf("chat_channels missing %s: %v", ch, structured[channelsOut](t, chans).Channels)
	}
}

func TestInboxAndMention(t *testing.T) {
	// claude-code opens a thread tagging @codex; codex's inbox should see it.
	// Both agents share one workspace.
	dir := newWorkspace(t)
	author := connectIn(t, dir, "claude-code")
	open := call(t, author, "chat_open", openIn{Channel: "#design", Body: "@codex review please", To: "codex"})
	_ = open

	codex := connectIn(t, dir, "codex")
	inbox := call(t, codex, "chat_inbox", inboxIn{Mark: true})
	msgs := structured[messagesOut](t, inbox).Messages
	if len(msgs) == 0 {
		t.Fatal("codex inbox empty; expected the @codex mention")
	}
	joined := strings.Join(msgs, "\n")
	if !strings.Contains(joined, "review please") {
		t.Fatalf("inbox missing mention: %v", msgs)
	}
}

func TestWhoami(t *testing.T) {
	cs := connect(t, "claude-code")
	res := call(t, cs, "whoami", struct{}{})
	who := structured[whoamiOut](t, res)
	if who.Agent != "claude-code" {
		t.Fatalf("whoami agent = %q, want claude-code", who.Agent)
	}
}

func TestKB(t *testing.T) {
	cs := connect(t, "claude-code")
	add := call(t, cs, "kb_add", kbAddIn{Type: "decision", Title: "Use streaming", Body: "lower memory", Tags: "perf,csv"})
	if structured[kbAddOut](t, add).ID == "" {
		t.Fatal("kb_add returned empty id")
	}
	got := call(t, cs, "kb_search", kbSearchIn{Tags: "perf"})
	if len(structured[kbSearchOut](t, got).Entries) == 0 {
		t.Fatal("kb_search found nothing for tag perf")
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
