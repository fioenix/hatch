//go:build hatch_legacy

package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/metrics"
	"github.com/fioenix/overclaud/hatch/internal/presence"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newWorkloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workload",
		Short: "Team load: presence, WIP, throughput, cycle time",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			rep, err := metrics.Compute(store.NewLedger(ws.Layout))
			if err != nil {
				return err
			}
			load := wipLoad(ws)
			pres := presence.Load(ws.Layout)
			out := cmd.OutOrStdout()
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "AGENT\tPRESENCE\tWIP\tDONE\tFLAG")
			for _, a := range ws.Registry.Agents {
				wip := fmt.Sprintf("%d", load[a.ID])
				if a.WIP > 0 {
					wip = fmt.Sprintf("%d/%d", load[a.ID], a.WIP)
				}
				done := 0
				if s := rep.Agents[a.ID]; s != nil {
					done = s.Done
				}
				flag := ""
				switch {
				case !pres.CanTakeWork(a.ID):
					flag = "off"
				case overWIP(a, load):
					flag = "OVERLOADED"
				case load[a.ID] == 0:
					flag = "idle"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n", a.ID, pres.StatusOf(a.ID), wip, done, flag)
			}
			tw.Flush()
			fmt.Fprintf(out, "\nthroughput: %d done · avg cycle: %s\n", rep.Throughput, rep.CycleAvg.Round(1e9))
			return nil
		},
	}
	return cmd
}

func newPerfCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perf [agent]",
		Short: "Per-agent operational scorecard (from the ledger)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			rep, err := metrics.Compute(store.NewLedger(ws.Layout))
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "AGENT\tCLAIMS\tDONE\tREVIEWS\tGATE-FAIL\tESCAL\tCOST")
			for _, s := range rep.Sorted() {
				if len(args) == 1 && s.Agent != args[0] {
					continue
				}
				fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d\t$%.4f\n",
					s.Agent, s.Claims, s.Done, s.Reviews, s.GateFails, s.Escalations, s.CostUSD)
			}
			tw.Flush()
			fmt.Fprintln(out, "\n(chỉ số vận hành, không phải đánh giá con người — xem docs/13 Goodhart)")
			return nil
		},
	}
	return cmd
}
