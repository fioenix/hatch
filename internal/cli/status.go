package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/hatch/internal/bus"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Read-only summary: open threads (tasks), recent activity, the roster",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			b := bus.New(ws.Layout)
			out := cmd.OutOrStdout()

			chans, err := b.Channels()
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "## Threads (tasks) — %d\n", len(chans))
			if len(chans) == 0 {
				fmt.Fprintln(out, "  (none yet — agents open threads through the Hatch MCP server)")
			} else {
				tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
				fmt.Fprintln(tw, "  THREAD\tMSGS\tLAST\tWHO")
				for _, ch := range chans {
					msgs, _ := b.Messages(ch)
					last, who := "", ""
					if n := len(msgs); n > 0 {
						who = msgs[n-1].From
						if t, e := time.Parse(time.RFC3339Nano, msgs[n-1].TS); e == nil {
							last = t.Format("01-02 15:04")
						}
					}
					fmt.Fprintf(tw, "  #%s\t%d\t%s\t%s\n", ch, len(msgs), last, who)
				}
				tw.Flush()
			}

			// Roster: who's on the squad and what roles they may hold.
			fmt.Fprintf(out, "\n## Roster — %d agents\n", len(ws.Registry.Agents))
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			for _, a := range ws.Registry.Agents {
				roles := "-"
				if len(a.Roles) > 0 {
					roles = fmt.Sprintf("%v", a.Roles)
				}
				fmt.Fprintf(tw, "  %s\t%s\t%s\n", a.ID, a.Kind, roles)
			}
			tw.Flush()
			fmt.Fprintln(out)
			return nil
		},
	}
	return cmd
}
