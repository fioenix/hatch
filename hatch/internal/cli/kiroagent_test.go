package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteKiroAgent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".kiro", "cli-agents", "hatch.json")

	if err := writeKiroAgent(p, "kiro"); err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	raw, _ := os.ReadFile(p)
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if root["name"] != "hatch" {
		t.Errorf("name = %v", root["name"])
	}
	if !strings.Contains(string(raw), "hatch brief --as kiro --format text") {
		t.Errorf("missing agentSpawn hook: %s", raw)
	}
	if !strings.Contains(string(raw), `"--as"`) || !strings.Contains(string(raw), "mcp") {
		t.Errorf("missing mcpServers.hatch: %s", raw)
	}

	// Idempotent: re-run does not duplicate the hook.
	if err := writeKiroAgent(p, "kiro"); err != nil {
		t.Fatal(err)
	}
	raw2, _ := os.ReadFile(p)
	if n := strings.Count(string(raw2), "hatch brief --as kiro"); n != 1 {
		t.Errorf("hook duplicated: count=%d", n)
	}

	// Preserves user-added fields.
	json.Unmarshal(raw2, &root)
	root["tools"] = []any{"fs_read"}
	b, _ := json.MarshalIndent(root, "", "  ")
	os.WriteFile(p, b, 0o644)
	if err := writeKiroAgent(p, "kiro"); err != nil {
		t.Fatal(err)
	}
	if raw3, _ := os.ReadFile(p); !strings.Contains(string(raw3), "fs_read") {
		t.Errorf("dropped user field 'tools': %s", raw3)
	}
}
