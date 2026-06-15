package cli

import (
	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/tui"
)

func newBoardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "board",
		Short: "Read-only mission-control TUI (threads + chat + ledger)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			_, err = tui.New(ws).Run()
			return err
		},
	}
	return cmd
}

func newChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Read-only Slack-style TUI for watching agent communication",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			_, err = tui.NewChat(ws).Run()
			return err
		},
	}
	return cmd
}
