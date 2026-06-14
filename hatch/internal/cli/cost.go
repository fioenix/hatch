package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newCostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost <ticket>",
		Short: "Total tracked cost (USD + tokens) for a ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			recs, err := store.NewLedger(ws.Layout).ScanCosts()
			if err != nil {
				return err
			}
			var usd float64
			var tokens int
			for _, r := range recs {
				if r.Ticket == args[0] {
					usd += r.USD
					tokens += r.Tokens
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: $%.4f · %d tokens\n", args[0], usd, tokens)
			return nil
		},
	}
	return cmd
}

func newBudgetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "Tracked spend per agent + team vs budget (no enforcement)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			recs, err := store.NewLedger(ws.Layout).ScanCosts()
			if err != nil {
				return err
			}
			spend := map[string]float64{}
			toks := map[string]int{}
			var total float64
			for _, r := range recs {
				spend[r.Agent] += r.USD
				toks[r.Agent] += r.Tokens
				total += r.USD
			}
			out := cmd.OutOrStdout()
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "AGENT\tSPEND\tBUDGET\tTOKENS\tUSE%")
			for _, a := range ws.Registry.Agents {
				budget := "-"
				pct := "-"
				if a.BudgetUSD > 0 {
					budget = fmt.Sprintf("$%.2f", a.BudgetUSD)
					pct = fmt.Sprintf("%.0f%%", spend[a.ID]/a.BudgetUSD*100)
				}
				fmt.Fprintf(tw, "%s\t$%.4f\t%s\t%d\t%s\n", a.ID, spend[a.ID], budget, toks[a.ID], pct)
			}
			tw.Flush()
			teamLine := fmt.Sprintf("\nTEAM: $%.4f", total)
			if ws.Registry.Policy.TeamBudgetUSD > 0 {
				teamLine += fmt.Sprintf(" / $%.2f (%.0f%%)", ws.Registry.Policy.TeamBudgetUSD, total/ws.Registry.Policy.TeamBudgetUSD*100)
			}
			fmt.Fprintln(out, teamLine)
			// track-only: warn, never block.
			for _, a := range ws.Registry.Agents {
				if a.BudgetUSD > 0 && spend[a.ID] >= 0.8*a.BudgetUSD {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s at %.0f%% of budget\n", a.ID, spend[a.ID]/a.BudgetUSD*100)
				}
			}
			return nil
		},
	}
	return cmd
}
