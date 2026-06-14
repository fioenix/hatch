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
