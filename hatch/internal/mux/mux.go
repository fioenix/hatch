//go:build hatch_legacy

// Package mux opens agent runs in terminal-multiplexer panes (tmux / Zellij)
// so you can watch each spawned agent live, side by side (observability tier B,
// docs/18). The single-process TUI (`hatch board`) is the no-deps default; this
// is for "many real panes".
package mux

import (
	"fmt"
	"os/exec"
	"strings"
)

// Kinds.
const (
	Tmux   = "tmux"
	Zellij = "zellij"
)

// Available reports whether a multiplexer binary is on PATH.
func Available(kind string) bool {
	_, err := exec.LookPath(kind)
	return err == nil
}

// Command builds (without running) the multiplexer invocation that opens a pane
// titled `title` running `inner` (argv). Exposed for testing/inspection.
func Command(kind, title string, inner []string) ([]string, error) {
	cmdline := shJoin(inner)
	switch kind {
	case Tmux:
		// Split the current window; falls back to a new window name.
		return []string{"tmux", "new-window", "-n", title, cmdline}, nil
	case Zellij:
		return []string{"zellij", "run", "--name", title, "--", inner[0]}, nil
	default:
		return nil, fmt.Errorf("unknown mux %q (tmux|zellij)", kind)
	}
}

// Launch opens a pane running inner in the given multiplexer.
func Launch(kind, title string, inner []string) error {
	if !Available(kind) {
		return fmt.Errorf("%s not found on PATH", kind)
	}
	var c *exec.Cmd
	switch kind {
	case Tmux:
		c = exec.Command("tmux", "new-window", "-n", title, shJoin(inner))
	case Zellij:
		args := append([]string{"run", "--name", title, "--"}, inner...)
		c = exec.Command("zellij", args...)
	default:
		return fmt.Errorf("unknown mux %q (tmux|zellij)", kind)
	}
	return c.Run()
}

// shJoin quotes argv into a single shell command string (for tmux's command arg).
func shJoin(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		if strings.ContainsAny(a, " \t\"'$`\\") {
			parts[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		} else {
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}
