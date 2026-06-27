package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWiringStatus(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)

	// Claude is always reported as plugin-wired.
	if mcp, hook := wiringStatus("claude", repo); mcp != "plugin" || hook != "plugin" {
		t.Errorf("claude: got %s/%s, want plugin/plugin", mcp, hook)
	}

	// Nothing wired yet → all ✗ (agy hook is n/a "—").
	if mcp, hook := wiringStatus("codex", repo); mcp != "✗" || hook != "✗" {
		t.Errorf("codex unwired: got %s/%s, want ✗/✗", mcp, hook)
	}
	if mcp, hook := wiringStatus("agy", repo); mcp != "✗" || hook != "✗" {
		t.Errorf("agy unwired: got %s/%s, want ✗/✗", mcp, hook)
	}

	// Wire codex (config.toml + hooks.json) and kiro (project mcp.json).
	mustWrite(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.hatch]\n")
	mustWrite(t, filepath.Join(home, ".codex", "hooks.json"), `{"x":"hatch brief --as codex"}`)
	mustWrite(t, filepath.Join(repo, ".kiro", "settings", "mcp.json"), `{"mcpServers":{"hatch":{}}}`)

	if mcp, hook := wiringStatus("codex", repo); mcp != "✓" || hook != "✓" {
		t.Errorf("codex wired: got %s/%s, want ✓/✓", mcp, hook)
	}
	if mcp, _ := wiringStatus("kiro", repo); mcp != "✓" {
		t.Errorf("kiro mcp wired: got %s, want ✓", mcp)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
