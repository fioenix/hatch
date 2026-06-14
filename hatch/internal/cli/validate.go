package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/store"
	"github.com/fioenix/overclaud/hatch/internal/validate"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check registry, workflow and board for consistency",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			probs := ws.Validate()
			probs = append(probs, validate.Board(ws, store.NewBoard(ws.Layout))...)
			if len(probs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "✓ workspace is valid")
				return nil
			}
			for _, p := range probs {
				fmt.Fprintln(cmd.OutOrStdout(), "✗ "+p.String())
			}
			return fmt.Errorf("%d problem(s) found", len(probs))
		},
	}
	return cmd
}
