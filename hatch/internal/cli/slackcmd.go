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
		Long: "Mirrors every squad message into one Slack channel and ingests your Slack messages " +
			"back onto the bus, where `hatch daemon` delivers them. Each agent maps to its own Slack " +
			"app/bot, so it posts and is @mentioned as a real Slack principal; @codex in Slack wakes " +
			"codex like any peer. This is a window into the room, not a controller.\n\n" +
			"Setup: one \"hub\" app with Socket Mode (scopes chat:write, chat:write.customize, " +
			"channels:history, connections:write; subscribe to message.channels) plus one app per agent " +
			"(scope chat:write), each invited to the channel. Config from HATCH_SLACK_{APP_TOKEN," +
			"BOT_TOKEN,CHANNEL,BOSS} + HATCH_SLACK_TOKEN_<AGENT>, or .hatch/slack/config.json " +
			"(gitignored — never commit tokens). Ctrl-C to stop.",
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
