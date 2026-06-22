package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/slack"
)

func newSlackCmd() *cobra.Command {
	var interval time.Duration
	var once, dryRun bool
	cmd := &cobra.Command{
		Use:   "slack",
		Short: "Bridge the squad chat to a Slack channel (mirror out, @tag in)",
		Long: "Mirrors every squad message into one Slack channel — each agent shown by its own " +
			"name and icon — and ingests your Slack messages back onto the bus, where `hatch daemon` " +
			"delivers them. Tag an agent from Slack (\"@codex …\") to wake it like any peer. This is a " +
			"window into the room, not a controller.\n\n" +
			"Tokens come from HATCH_SLACK_{BOT_TOKEN,APP_TOKEN,CHANNEL,BOSS} or .hatch/slack/config.json " +
			"(gitignored — never commit them). The Slack app needs scopes chat:write, chat:write.customize " +
			"and connections:write, with Socket Mode enabled. Ctrl-C to stop.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			return slack.Run(ws.Layout, slack.Options{
				Interval: interval,
				Once:     once,
				DryRun:   dryRun,
				Stdout:   cmd.OutOrStdout(),
			})
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "how often to mirror new bus messages to Slack")
	cmd.Flags().BoolVar(&once, "once", false, "mirror the current backlog once and exit (no Socket Mode)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the mirror to stdout instead of Slack (needs no tokens)")
	return cmd
}
