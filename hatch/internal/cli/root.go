// Package cli wires the `hatch` command tree.
package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// NewRoot builds the root command with all subcommands attached.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "hatch",
		Short:         "Hatch — a multi-agent coding squad on the filesystem",
		Long:          "Hatch orchestrates multiple coding-agent CLIs as one squad: a single source of truth compiled per agent, a file-based board, an append-only ledger, and a shared knowledge base.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}
	// The embedded-harness command set: SSOT compile, the MCP server agents
	// drive themselves through, read-only views over the shared chat + ledger,
	// and the knowledge base. Self-driving operator commands (run/plan/watch,
	// ceremonies, tickets, …) are archived behind the `hatch_legacy` build tag.
	root.AddCommand(
		newSetupCmd(),
		newInitCmd(),
		newBriefCmd(),
		newGuardCmd(),
		newTraceCmd(),
		newCompileCmd(),
		newValidateCmd(),
		newStatusCmd(),
		newRosterCmd(),
		newDaemonCmd(),
		newSlackCmd(),
		newSessionsCmd(),
		newKBCmd(),
		newBoardCmd(),
		newChatCmd(),
		newSyncCmd(),
		newHookCmd(),
		newMsgCmd(),
		newChannelCmd(),
		newInboxCmd(),
		newThreadCmd(),
		newSearchCmd(),
		newDocCmd(),
		newLogsCmd(),
		newOrgCmd(),
		newDoctorCmd(),
		newMCPCmd(),
	)
	addLegacyCommands(root)
	return root
}

// loadWorkspace resolves the workspace with global+local layering: a local
// .hatch (nearest ancestor of cwd) overrides the global ~/.hatch. Compiled
// outputs always target the current repo (cwd) — for a local workspace that is
// the repo root; for the global default it is wherever you're working.
func loadWorkspace() (*config.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if l, err := paths.FindLocal(cwd); err == nil {
		ws, err := config.Load(l)
		if err != nil {
			return nil, err
		}
		ws.OutputRoot = l.RepoRoot()
		return ws, nil
	}
	if g := paths.GlobalRoot(); g != "" {
		if fi, statErr := os.Stat(g); statErr == nil && fi.IsDir() {
			ws, err := config.Load(paths.Layout{Root: g})
			if err != nil {
				return nil, err
			}
			ws.OutputRoot = cwd // global SSOT compiles into the current repo
			return ws, nil
		}
	}
	return nil, paths.ErrNotFound
}
