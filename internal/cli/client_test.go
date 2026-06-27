package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveClientKind(t *testing.T) {
	cases := map[string]string{
		"cc": "claude", "claude": "claude", "claude-code": "claude",
		"codex": "codex", "agy": "agy", "antigravity": "agy",
		"kiro": "kiro", "kiro-cli": "kiro",
	}
	for alias, want := range cases {
		got, ok := resolveClientKind(alias)
		if !ok || got != want {
			t.Errorf("resolveClientKind(%q) = %q,%v; want %q", alias, got, ok, want)
		}
	}
	if _, ok := resolveClientKind("gpt"); ok {
		t.Error("resolveClientKind(gpt) should be unknown")
	}
}

func TestWriteServerJSONMergePreserves(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nested", "mcp_config.json")
	// Seed with another server + an unrelated key.
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"mcpServers":{"other":{"command":"x"}},"theme":"dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeServerJSON(p, "agy", false); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	raw, _ := os.ReadFile(p)
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	servers := got["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Error("merge dropped pre-existing 'other' server")
	}
	h, ok := servers["hatch"].(map[string]any)
	if !ok {
		t.Fatal("merge did not add 'hatch' server")
	}
	if h["command"] != "hatch" {
		t.Errorf("hatch command = %v, want hatch", h["command"])
	}
	if got["theme"] != "dark" {
		t.Error("merge dropped unrelated top-level 'theme' key")
	}
}

func TestWriteServerJSONDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "mcp.json")
	if err := writeServerJSON(p, "codex", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("dry-run should not create %s", p)
	}
}
