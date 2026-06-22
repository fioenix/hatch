package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/session"
)

func newSessionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Show agents' resumable sessions, one per (agent, thread)",
		Long: "Lists the warm CLI sessions the wake daemon resumes per task thread — which agent, " +
			"which bus thread, the session id, status (live/stale) and when it was last resumed. " +
			"The bus + KB remain the source of truth; a session is just a warm cache, so a stale one " +
			"simply starts fresh on the next wake. Agents whose CLI exposes no session id (e.g. agy) " +
			"run stateless and do not appear here.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			all := session.New(ws.Layout).All()
			fmt.Fprintf(out, "## Sessions — %d\n", len(all))
			if len(all) == 0 {
				fmt.Fprintln(out, "  (none yet — created when the daemon wakes claude/codex on a thread)")
				return nil
			}
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "  AGENT\tTHREAD\tKIND\tSTATUS\tSESSION\tLAST-RESUMED")
			for _, s := range all {
				id := s.ID
				if len(id) > 8 {
					id = id[:8]
				}
				if id == "" {
					id = "-"
				}
				last := s.LastResumedAt
				if t, e := time.Parse(time.RFC3339, s.LastResumedAt); e == nil {
					last = t.Format("01-02 15:04")
				}
				fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%s\n", s.Agent, s.Thread, s.Kind, s.Status, id, last)
			}
			tw.Flush()
			fmt.Fprintln(out)
			return nil
		},
	}
	return cmd
}
