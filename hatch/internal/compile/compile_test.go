package compile

import (
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
	// default roster touches claude, codex, gemini, kiro → 4 surfaces.
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
