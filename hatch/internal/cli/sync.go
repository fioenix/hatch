package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/compile"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newSyncCmd() *cobra.Command {
	var fix bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile derived state: ticket status↔lane, KB index, compile freshness",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			b := store.NewBoard(ws.Layout)

			// 1. ticket status must match the lane it lives in.
			tickets, err := b.List(ws.Workflow.LaneIDs())
			if err != nil {
				return err
			}
			drift := 0
			for _, t := range tickets {
				if t.Status != t.Lane {
					drift++
					if fix {
						t.Status = t.Lane
						if _, err := b.Write(t); err != nil {
							return err
						}
						fmt.Fprintf(out, "fixed %s status → %s\n", t.ID, t.Lane)
					} else {
						fmt.Fprintf(out, "drift: %s status=%q lane=%q\n", t.ID, t.Status, t.Lane)
					}
				}
			}

			// 2. KB index rebuilt from entries.
			if err := store.NewKB(ws.Layout).RebuildIndex(); err != nil {
				return err
			}
			fmt.Fprintln(out, "kb/index.md rebuilt")

			// 3. compile freshness (report only; never auto-edit outputs here).
			reason, err := compile.StaleReason(ws.Layout, ws.Layout.RepoRoot())
			if err != nil {
				return err
			}
			if reason != "" {
				fmt.Fprintf(out, "compiled files stale: %s (run `hatch compile`)\n", reason)
			} else {
				fmt.Fprintln(out, "compiled files up to date")
			}

			if drift > 0 && !fix {
				return fmt.Errorf("%d status drift(s) found; re-run with --fix", drift)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "apply safe fixes (status alignment)")
	return cmd
}
