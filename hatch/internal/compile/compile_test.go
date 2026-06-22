package compile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
)

func TestRunProducesSurfaces(t *testing.T) {
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := config.Load(l)
	if err != nil {
		t.Fatal(err)
	}
	res, warnings, err := Run(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	// default roster touches claude, codex, agy, kiro → 4 surfaces.
	if len(res.Bundles) != 4 {
		t.Fatalf("expected 4 surfaces, got %d", len(res.Bundles))
	}
	for _, f := range []string{"CLAUDE.md", "AGENTS.md", "GEMINI.md", ".kiro/steering/hatch.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing surface %s: %v", f, err)
		}
	}
	// Claude surface must not be hand-editable and must carry layering headers.
	claude, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	for _, want := range []string{"DO NOT EDIT", "Mission (L0)", "Your roles (L1)", "Context map (L2"} {
		if !strings.Contains(string(claude), want) {
			t.Errorf("CLAUDE.md missing %q", want)
		}
	}
}

func TestCompileInjectsProtocolAndMCP(t *testing.T) {
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, err := config.Load(l)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Run(ws); err != nil {
		t.Fatal(err)
	}

	// Lead surface (claude-code holds conductor) carries the orchestrator block,
	// the workflow prose, the chat protocol and the DoD self-check.
	claude, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	for _, want := range []string{
		"Conductor (orchestrator)",
		"Workflow — scrum",
		"Chat protocol",
		"chat_open",
		"Definition of Done",
		"make test",
	} {
		if !strings.Contains(string(claude), want) {
			t.Errorf("CLAUDE.md missing %q", want)
		}
	}

	// A non-lead surface gets the protocol but NOT the orchestrator block.
	agents, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if strings.Contains(string(agents), "Conductor (orchestrator)") {
		t.Error("AGENTS.md should not carry the orchestrator block")
	}
	if !strings.Contains(string(agents), "Chat protocol") {
		t.Error("AGENTS.md missing chat protocol")
	}

	// MCP registration: kiro's repo config plus a Codex paste-snippet. Claude is
	// NOT given a .mcp.json — it is wired by its user-global plugin (hatch setup).
	if _, err := os.Stat(filepath.Join(dir, ".mcp.json")); !os.IsNotExist(err) {
		t.Errorf(".mcp.json should not be written (claude uses the plugin); err=%v", err)
	}
	kmcp, err := os.ReadFile(filepath.Join(dir, ".kiro", "settings", "mcp.json"))
	if err != nil {
		t.Fatalf("missing kiro MCP config: %v", err)
	}
	if !strings.Contains(string(kmcp), `"hatch"`) || !strings.Contains(string(kmcp), "kiro") {
		t.Errorf("kiro mcp.json missing hatch server / kiro identity: %s", kmcp)
	}
	if _, err := os.Stat(filepath.Join(dir, ".hatch", "run", "mcp", "codex.codex.toml")); err != nil {
		t.Errorf("missing codex MCP snippet: %v", err)
	}
}

func TestMergeJSONServerPreservesOthers(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".mcp.json")
	seed := `{"mcpServers":{"other":{"command":"x"}},"extra":true}`
	if err := os.WriteFile(p, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mergeJSONServer(p, "claude-code"); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	raw, _ := os.ReadFile(p)
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	servers := got["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Error("merge dropped the pre-existing 'other' server")
	}
	if _, ok := servers["hatch"]; !ok {
		t.Error("merge did not add the 'hatch' server")
	}
	if got["extra"] != true {
		t.Error("merge dropped the top-level 'extra' key")
	}
}

func TestStaleDetection(t *testing.T) {
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	if _, _, err := Run(ws); err != nil {
		t.Fatal(err)
	}
	if reason, _ := StaleReason(l, dir); reason != "" {
		t.Fatalf("expected fresh, got stale: %s", reason)
	}
	// Mutating the SSOT makes outputs stale.
	f := l.Charter()
	data, _ := os.ReadFile(f)
	os.WriteFile(f, append(data, []byte("\nchanged\n")...), 0o644)
	if reason, _ := StaleReason(l, dir); reason == "" {
		t.Fatal("expected stale after editing charter")
	}
}
