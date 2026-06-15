//go:build hatch_legacy

package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// TestExecuteRunsMockAgentEndToEnd really spawns a process (a tiny shell script
// standing in for hatch-mock), exercising the full execute + capture + ledger
// path without a live agent CLI.
func TestExecuteRunsMockAgentEndToEnd(t *testing.T) {
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)

	// A fake agent binary: echoes its args (incl the prompt) and exits 0.
	script := filepath.Join(dir, "mock.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"MOCKREPLY $*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	agent := model.Agent{ID: "codex", Kind: "mock", Cmd: script}

	o := Orchestrator{Ledger: store.NewLedger(l), Bus: bus.New(l)}
	out, err := o.Execute(ws, agent, "T-1", "implement the export", RunOptions{Stdout: io_discard{}})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Executed || out.ExitCode != 0 {
		t.Fatalf("expected executed exit 0, got executed=%v code=%d err=%v", out.Executed, out.ExitCode, out.Err)
	}
	if !strings.Contains(out.Output, "MOCKREPLY") || !strings.Contains(out.Output, "implement the export") {
		t.Fatalf("prompt not passed/captured: %q", out.Output)
	}
	// Ledger should have recorded start + progress for the run.
	files, _ := store.NewLedger(l).Files()
	if len(files) == 0 {
		t.Fatal("expected ledger entries from the run")
	}
	raw, _ := os.ReadFile(files[0])
	if !strings.Contains(string(raw), "T-1") {
		t.Fatalf("ledger missing run entry: %s", raw)
	}
}

type io_discard struct{}

func (io_discard) Write(p []byte) (int, error) { return len(p), nil }
