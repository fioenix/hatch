package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fioenix/hatch/internal/compile"
	"github.com/fioenix/hatch/internal/store"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile derived state: KB index and compile freshness",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()

			// 1. KB index rebuilt from entries.
			if err := store.NewKB(ws.Layout).RebuildIndex(); err != nil {
				return err
			}
			fmt.Fprintln(out, "kb/index.md rebuilt")

			// 2. compile freshness (report only; never auto-edit outputs here).
			reason, err := compile.StaleReason(ws.Layout, ws.Layout.RepoRoot())
			if err != nil {
				return err
			}
			if reason != "" {
				fmt.Fprintf(out, "compiled files stale: %s (run `hatch compile`)\n", reason)
			} else {
				fmt.Fprintln(out, "compiled files up to date")
			}
			return nil
		},
	}
	return cmd
}
