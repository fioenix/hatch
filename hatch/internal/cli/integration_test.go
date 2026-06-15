//go:build hatch_legacy

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

// TestLifecycleEndToEnd drives the real CLI through a full ticket lifecycle in
// a temp workspace, with a shell script standing in for an agent CLI — proving
// the command wiring works end to end (init → compile → ticket → run → logs).
func TestLifecycleEndToEnd(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if out, err := run(t, "init", "-w", "scrum"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// Wire the codex agent to a mock script so `run` really spawns + captures.
	mock := filepath.Join(dir, "mock.sh")
	if err := os.WriteFile(mock, []byte("#!/bin/sh\necho \"MOCKREPLY $*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	regPath := filepath.Join(dir, ".hatch", "registry.yaml")
	reg, _ := os.ReadFile(regPath)
	reg = bytes.Replace(reg, []byte("    kind: codex\n"),
		[]byte("    kind: mock\n    cmd: "+mock+"\n"), 1)
	if err := os.WriteFile(regPath, reg, 0o644); err != nil {
		t.Fatal(err)
	}

	if out, err := run(t, "compile"); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Fatalf("compile did not produce CLAUDE.md: %v", err)
	}

	if out, err := run(t, "validate"); err != nil {
		t.Fatalf("validate: %v\n%s", err, out)
	}

	if out, err := run(t, "ticket", "new", "--title", "Export CSV", "--role", "implementer", "--priority", "P1"); err != nil {
		t.Fatalf("ticket new: %v\n%s", err, out)
	}
	if out, err := run(t, "ticket", "claim", "T-001", "--agent", "codex", "--why", "test"); err != nil {
		t.Fatalf("claim: %v\n%s", err, out)
	}
	if out, err := run(t, "run", "T-001", "--agent", "codex"); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}

	logs, err := run(t, "logs", "T-001")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if !strings.Contains(logs, "MOCKREPLY") {
		t.Fatalf("transcript missing agent output:\n%s", logs)
	}

	report, err := run(t, "report")
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.Contains(report, "Status report") || !strings.Contains(report, "Budget") {
		t.Fatalf("report malformed:\n%s", report)
	}
}
