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
	root.AddCommand(
		newInitCmd(),
		newCompileCmd(),
		newValidateCmd(),
		newStatusCmd(),
		newStandupCmd(),
		newTicketCmd(),
		newKBCmd(),
		newGateCmd(),
		newRunCmd(),
		newPlanCmd(),
		newWatchCmd(),
		newBoardCmd(),
		newSyncCmd(),
		newHookCmd(),
		newMsgCmd(),
		newChannelCmd(),
		newInboxCmd(),
		newThreadCmd(),
		newAskCmd(),
		newConveneCmd(),
		newSearchCmd(),
		newCeremonyCmd(),
		newEscalateCmd(),
		newPairCmd(),
		newMobCmd(),
		newPresenceCmd(),
		newOncallCmd(),
		newCostCmd(),
		newBudgetCmd(),
		newWorkloadCmd(),
		newPerfCmd(),
		newDocCmd(),
	)
	return root
}

// loadWorkspace finds the .hatch workspace from cwd and loads its config.
func loadWorkspace() (*config.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	l, err := paths.Find(cwd)
	if err != nil {
		return nil, err
	}
	return config.Load(l)
}
