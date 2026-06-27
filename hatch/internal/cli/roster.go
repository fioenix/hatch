package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/roster"
)

func newRosterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roster",
		Short: "Who is in the room: live presence of agents that have joined the workspace",
		Long: "Shows the live workspace room — members that joined via the Hatch MCP `join` tool, " +
			"their roles, reachability (online/idle/suspended/offline) and last-seen. " +
			"This is the team-simulation presence layer, distinct from the static registry shown by `hatch status`.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			r, err := roster.New(ws.Layout).Effective(time.Now())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "## Room — %d members\n", len(r))
			if len(r) == 0 {
				fmt.Fprintln(out, "  (empty — agents join through the Hatch MCP `join` tool)")
				return nil
			}
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "  MEMBER\tKIND\tROLES\tSTATUS\tSESSION\tLAST-SEEN")
			for _, m := range roster.Members(r) {
				roles := "-"
				if len(m.Roles) > 0 {
					roles = fmt.Sprintf("%v", m.Roles)
				}
				session := "-"
				if m.SessionID != "" {
					session = "yes"
				}
				last := m.LastSeen
				if t, e := time.Parse(time.RFC3339, m.LastSeen); e == nil {
					last = t.Format("01-02 15:04")
				}
				fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%s\n", m.ID, m.Kind, roles, m.Status, session, last)
			}
			tw.Flush()
			fmt.Fprintln(out)
			return nil
		},
	}
	return cmd
}
