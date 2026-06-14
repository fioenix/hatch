package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the board: tickets per lane, assignees, WIP",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			b := store.NewBoard(ws.Layout)
			out := cmd.OutOrStdout()

			for _, lane := range ws.Workflow.Lanes {
				tickets, err := b.ListLane(lane.ID)
				if err != nil {
					return err
				}
				limit := ""
				if lane.WIPLimit > 0 {
					limit = fmt.Sprintf(" (WIP %d/%d)", len(tickets), lane.WIPLimit)
				}
				fmt.Fprintf(out, "\n## %s — %d%s\n", lane.ID, len(tickets), limit)
				if len(tickets) == 0 {
					continue
				}
				tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
				for _, t := range tickets {
					assignee := t.Assignee
					if assignee == "" {
						assignee = "-"
					}
					prio := t.Priority
					if prio == "" {
						prio = "--"
					}
					fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n", t.ID, prio, t.Role, assignee, t.Title)
				}
				tw.Flush()
			}
			fmt.Fprintln(out)
			printWIPWarnings(cmd, ws, b)
			return nil
		},
	}
	return cmd
}

// printWIPWarnings flags agents holding more in-flight tickets than their limit.
func printWIPWarnings(cmd *cobra.Command, ws *config.Workspace, b *store.Board) {
	tickets, err := b.List(ws.Workflow.LaneIDs())
	if err != nil {
		return
	}
	perAgent := map[string]int{}
	for _, t := range tickets {
		if t.Assignee != "" && t.Lane != "done" {
			perAgent[t.Assignee]++
		}
	}
	for _, a := range ws.Registry.Agents {
		if a.WIP > 0 && perAgent[a.ID] > a.WIP {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: agent %s over WIP (%d/%d)\n", a.ID, perAgent[a.ID], a.WIP)
		}
	}
}
