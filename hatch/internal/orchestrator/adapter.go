// Package orchestrator (Phase 3) spawns coding-agent CLIs headlessly to work
// tickets, capturing their output to the ledger. Adapters translate a generic
// run request into the native invocation each agent expects; see
// docs/10-agent-adapters.md for the source of these mappings.
package orchestrator

import (
	"os/exec"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

// RunRequest is a normalized request to run one agent on one ticket.
type RunRequest struct {
	Agent    model.Agent
	Ticket   model.Ticket
	Prompt   string // the task prompt handed to the agent
	WorkDir  string // working directory (a git worktree)
	RepoRoot string // repository root (for context)
}

// Invocation is the concrete command an adapter would run, kept separate from
// execution so it can be shown in --dry-run and recorded for audit.
type Invocation struct {
	Args     []string // argv (Args[0] is the program)
	Env      []string // extra environment, KEY=VALUE
	StdinStr string   // data piped to stdin, if any
	Headless bool     // false ⇒ agent has no headless mode (manual handoff)
	Note     string   // explanation when not headless
}

// Adapter builds invocations for a given agent kind.
type Adapter interface {
	Kind() string
	// Build returns the invocation for a request.
	Build(req RunRequest) Invocation
}

// program returns the executable for an agent, honoring an explicit cmd.
func program(a model.Agent, def string) string {
	if a.Cmd != "" {
		return a.Cmd
	}
	return def
}

// adapters maps a kind to its adapter.
var adapters = map[string]Adapter{
	"claude":      claudeAdapter{},
	"codex":       codexAdapter{},
	"agy":         agyAdapter{},
	"kiro":        kiroAdapter{},
	"antigravity": manualAdapter{kind: "antigravity", reason: "Antigravity is IDE-driven; no confirmed headless CLI"},
	"mock":        mockAdapter{},
	"manual":      manualAdapter{kind: "manual", reason: "manual agent"},
	"shell":       manualAdapter{kind: "shell", reason: "shell agent has no standard headless contract"},
}

// AdapterFor returns the adapter for an agent kind, defaulting to manual.
func AdapterFor(kind string) Adapter {
	if a, ok := adapters[kind]; ok {
		return a
	}
	return manualAdapter{kind: kind, reason: "unknown kind"}
}

// Available reports whether an adapter's program is on PATH (manual ⇒ true).
func Available(a model.Agent) bool {
	inv := AdapterFor(a.Kind).Build(RunRequest{Agent: a})
	if !inv.Headless || len(inv.Args) == 0 {
		return true
	}
	_, err := exec.LookPath(inv.Args[0])
	return err == nil
}
