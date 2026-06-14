package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/metrics"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newReportCmd() *cobra.Command {
	var post bool
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Executive status summary (board, throughput, budget, risks, decisions)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			rep, err := buildReport(ws)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), rep)
			if post {
				if _, err := bus.New(ws.Layout).Post(bus.Message{
					Channel: "#leadership", From: "human:facilitator", To: []string{"*"}, Body: rep,
				}); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "\n(posted to #leadership)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&post, "post", false, "post the report to #leadership")
	return cmd
}

func buildReport(ws *config.Workspace) (string, error) {
	b := store.NewBoard(ws.Layout)
	var sb strings.Builder
	project := ws.Registry.Project
	if project == "" {
		project = "Hatch"
	}
	fmt.Fprintf(&sb, "# Status report — %s — %s\n\n", project, time.Now().Format("2006-01-02"))

	// Board.
	sb.WriteString("## Board\n")
	for _, lane := range ws.Workflow.Lanes {
		ts, err := b.ListLane(lane.ID)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "- %s: %d\n", lane.ID, len(ts))
	}

	// Throughput.
	m, err := metrics.Compute(store.NewLedger(ws.Layout))
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&sb, "\n## Throughput\n- done: %d · avg cycle: %s\n", m.Throughput, m.CycleAvg.Round(time.Second))

	// Budget.
	recs, _ := store.NewLedger(ws.Layout).ScanCosts()
	var total float64
	for _, r := range recs {
		total += r.USD
	}
	sb.WriteString("\n## Budget\n")
	if cap := ws.Registry.Policy.TeamBudgetUSD; cap > 0 {
		fmt.Fprintf(&sb, "- spend: $%.2f / $%.2f (%.0f%%)\n", total, cap, total/cap*100)
	} else {
		fmt.Fprintf(&sb, "- spend: $%.2f\n", total)
	}

	// Risks: open external blockers + escalations.
	sb.WriteString("\n## Risks\n")
	risk := 0
	for _, lane := range ws.Workflow.LaneIDs() {
		ts, _ := b.ListLane(lane)
		for _, t := range ts {
			for _, e := range t.OpenExternal() {
				fmt.Fprintf(&sb, "- %s blocked-external: %s (owner %s, eta %s)\n", t.ID, e.What, e.Owner, e.ETA)
				risk++
			}
		}
	}
	escal := 0
	for _, s := range m.Agents {
		escal += s.Escalations
	}
	if escal > 0 {
		fmt.Fprintf(&sb, "- escalations: %d\n", escal)
		risk++
	}
	if risk == 0 {
		sb.WriteString("- none\n")
	}

	// Recent decisions (ADRs).
	sb.WriteString("\n## Recent decisions\n")
	entries, _ := store.NewKB(ws.Layout).List()
	n := 0
	for i := len(entries) - 1; i >= 0 && n < 5; i-- {
		if entries[i].Type == model.KBDecision {
			fmt.Fprintf(&sb, "- %s %s\n", entries[i].ID, entries[i].Title)
			n++
		}
	}
	if n == 0 {
		sb.WriteString("- none\n")
	}
	return sb.String(), nil
}
