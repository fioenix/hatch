package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/gate"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newGateCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "gate <id>",
		Short: "Evaluate the gates for a ticket's transition without moving it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if to == "" {
				return fmt.Errorf("--to is required")
			}
			b := store.NewBoard(ws.Layout)
			t, ok, err := b.Find(args[0], ws.Workflow.LaneIDs())
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("ticket %s not found", args[0])
			}
			tr, ok := ws.Workflow.FindTransition(t.Lane, to)
			if !ok {
				return fmt.Errorf("no transition %s → %s", t.Lane, to)
			}
			if len(tr.Gates) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no gates on %s → %s\n", t.Lane, to)
				return nil
			}
			out := cmd.OutOrStdout()
			failed := 0
			for _, o := range gate.EvaluateAll(ws, tr.Gates, t, ws.Layout.RepoRoot()) {
				status := "✓"
				switch {
				case o.Human:
					status = "⊙ human"
				case !o.Passed:
					status = "✗"
					failed++
				}
				fmt.Fprintf(out, "  %s  %-16s %s\n", status, o.Name, o.Detail)
			}
			if failed > 0 {
				return fmt.Errorf("%d gate(s) failed", failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "target lane to evaluate gates for (required)")
	return cmd
}
