package cli

import (
	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/tui"
)

func newBoardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "board",
		Short: "Interactive TUI board dashboard",
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
	var as string
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Slack-style TUI for agent communication (channels, messages, compose)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			_, err = tui.NewChat(ws, as).Run()
			return err
		},
	}
	cmd.Flags().StringVar(&as, "as", "human:operator", "identity to post messages as")
	return cmd
}
