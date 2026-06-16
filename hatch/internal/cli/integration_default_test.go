//go:build !hatch_legacy

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run executes the hatch root command with args in the current dir, returning
// combined output. A fresh root is built each call (cobra is stateful).
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := NewRoot()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// TestEmbeddedHarnessFlow drives the embedded-harness command set end to end in
// a temp workspace: init → compile (protocol + MCP registration) → post a chat
// thread → read it back through the read-only views.
func TestEmbeddedHarnessFlow(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// --local: create a project .hatch in this repo (the default now targets
	// the global ~/.hatch).
	if out, err := run(t, "init", "--local", "-w", "scrum"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	if out, err := run(t, "compile"); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	claude, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("compile did not produce CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(claude), "Chat protocol") {
		t.Error("CLAUDE.md missing the chat protocol")
	}
	// Claude is wired by its plugin (hatch setup), not a repo .mcp.json. compile
	// registers MCP for the repo-only client: kiro's .kiro/settings/mcp.json.
	if _, err := os.Stat(filepath.Join(dir, ".mcp.json")); !os.IsNotExist(err) {
		t.Errorf(".mcp.json should not be written (claude uses the plugin); err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".kiro", "settings", "mcp.json")); err != nil {
		t.Errorf("compile did not register the kiro MCP server: %v", err)
	}

	// A chat thread is a task. Post one as a human operator.
	if out, err := run(t, "msg", "--from", "human:operator", "--channel", "#export-csv",
		"@codex please stream the CSV"); err != nil {
		t.Fatalf("msg: %v\n%s", err, out)
	}

	// The read-only views surface it.
	thread, err := run(t, "thread", "#export-csv")
	if err != nil {
		t.Fatalf("thread: %v", err)
	}
	if !strings.Contains(thread, "stream the CSV") {
		t.Fatalf("thread missing the message:\n%s", thread)
	}

	status, err := run(t, "status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(status, "export-csv") || !strings.Contains(status, "claude-code") {
		t.Fatalf("status should list the thread and roster:\n%s", status)
	}

	search, err := run(t, "search", "stream")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(search, "stream the CSV") {
		t.Fatalf("search did not recall the message:\n%s", search)
	}
}
