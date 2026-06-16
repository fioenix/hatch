package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeSessionStartHook(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hooks.json")
	// Seed an existing SessionStart hook (like a real ~/.codex/hooks.json).
	seed := `{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"node existing.js","timeout":30}]}],"Stop":[{"hooks":[{"type":"command","command":"node existing.js"}]}]}}`
	if err := os.WriteFile(p, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	added, err := mergeSessionStartHook(p, "hatch brief --as codex")
	if err != nil || !added {
		t.Fatalf("first merge: added=%v err=%v", added, err)
	}

	var root map[string]any
	raw, _ := os.ReadFile(p)
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	groups := root["hooks"].(map[string]any)["SessionStart"].([]any)
	if len(groups) != 2 {
		t.Fatalf("want 2 SessionStart groups (existing + ours), got %d", len(groups))
	}
	// Existing hook + the Stop hook must be preserved.
	if !strings.Contains(string(raw), "node existing.js") || !strings.Contains(string(raw), `"Stop"`) {
		t.Errorf("merge dropped existing hooks: %s", raw)
	}
	if !strings.Contains(string(raw), "hatch brief --as codex") {
		t.Errorf("merge missing our hook: %s", raw)
	}

	// Idempotent: second merge is a no-op.
	added, err = mergeSessionStartHook(p, "hatch brief --as codex")
	if err != nil || added {
		t.Fatalf("second merge should be no-op: added=%v err=%v", added, err)
	}

	// Creates the file when absent.
	p2 := filepath.Join(dir, "new", "hooks.json")
	if added, err := mergeSessionStartHook(p2, "hatch brief"); err != nil || !added {
		t.Fatalf("create-when-absent: added=%v err=%v", added, err)
	}
}

func TestWriteAgyHook(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hooks.json")
	// Seed an existing named hook — must be preserved.
	if err := os.WriteFile(p, []byte(`{"my-linter":{"PostToolUse":[{"matcher":"*","hooks":[{"command":"lint.sh"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeAgyHook(p, "hatch brief --as agy --format agy"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if root["my-linter"] == nil {
		t.Errorf("dropped existing hook: %s", raw)
	}
	if !strings.Contains(string(raw), "PreInvocation") || !strings.Contains(string(raw), "hatch brief --as agy") {
		t.Errorf("missing hatch PreInvocation hook: %s", raw)
	}
}
