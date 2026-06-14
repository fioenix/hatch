package orchestrator

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
)

func TestCommContextGathersInboxAndRecall(t *testing.T) {
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := config.Load(l)
	if err != nil {
		t.Fatal(err)
	}
	b := bus.New(l)
	// codex subscribes to #design and there's relevant chatter + a direct mention.
	b.Subscribe("#design", "codex")
	b.Post(bus.Message{Channel: "#design", From: "claude-code", To: []string{"#design"}, Body: "Export nên dùng CSV streaming"})
	b.Post(bus.Message{Channel: "T-001", From: "claude-code", To: []string{"codex"}, Type: bus.TypeAsk, Body: "@codex bắt đầu Export CSV nhé"})

	codex, _ := ws.Registry.AgentByID("codex")
	got := commContext(ws, codex, "Export CSV")
	if got == "" {
		t.Fatal("expected non-empty comm context")
	}
	if !strings.Contains(got, "Inbox") || !strings.Contains(got, "bắt đầu Export CSV") {
		t.Errorf("inbox mention missing:\n%s", got)
	}
	if !strings.Contains(got, "streaming") {
		t.Errorf("relevant recall missing:\n%s", got)
	}
}

func TestCommContextEmptyWhenNothing(t *testing.T) {
	dir := t.TempDir()
	l, _, _ := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	ws, _ := config.Load(l)
	codex, _ := ws.Registry.AgentByID("codex")
	if got := commContext(ws, codex, "anything"); got != "" {
		t.Fatalf("expected empty comm context, got: %s", got)
	}
}
