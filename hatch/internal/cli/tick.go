package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newTickCmd() *cobra.Command {
	var dryRun bool
	var max, parallel int
	cmd := &cobra.Command{
		Use:   "tick",
		Short: "One heartbeat: standup digest + dispatch claimable work + budget check (for cron/CI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()

			// 1. Standup digest → #standup.
			sr, err := ceremonyService(ws).Standup(ws, 1)
			if err != nil {
				return err
			}
			if !dryRun {
				_, _ = bus.New(ws.Layout).Post(bus.Message{
					Channel: "#standup", From: "human:facilitator", To: []string{"*"}, Body: sr.Markdown,
				})
			}
			fmt.Fprintln(out, "● standup digest posted")

			// 2. Dispatch claimable work.
			fmt.Fprintln(out, "● dispatch:")
			n, err := dispatchBacklog(ws, out, dispatchOpts{DryRun: dryRun, Max: max, Parallel: parallel})
			if err != nil {
				return err
			}

			// 3. Budget check (track-only warnings).
			recs, _ := store.NewLedger(ws.Layout).ScanCosts()
			spend := map[string]float64{}
			for _, r := range recs {
				spend[r.Agent] += r.USD
			}
			for _, a := range ws.Registry.Agents {
				if a.BudgetUSD > 0 && spend[a.ID] >= 0.8*a.BudgetUSD {
					fmt.Fprintf(out, "● budget warning: %s at %.0f%%\n", a.ID, spend[a.ID]/a.BudgetUSD*100)
				}
			}
			fmt.Fprintf(out, "tick done: %d dispatched\n", n)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without posting/claiming/running")
	cmd.Flags().IntVar(&max, "max", 5, "max tickets to dispatch this tick (0 = all)")
	cmd.Flags().IntVar(&parallel, "parallel", 1, "concurrent run workers")
	return cmd
}
