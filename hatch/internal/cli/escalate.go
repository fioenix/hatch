package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/store"
	"github.com/fioenix/overclaud/hatch/internal/wf"
)

func newEscalateCmd() *cobra.Command {
	var from, why string
	cmd := &cobra.Command{
		Use:   "escalate <ticket>",
		Short: "Escalate a ticket to the senior/on-call target (ledger + #escalations)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if why == "" {
				return fmt.Errorf("--why is required")
			}
			if err := wf.Escalate(ws, store.NewBoard(ws.Layout), store.NewLedger(ws.Layout), args[0], from, why); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "escalated %s → %s\n", args[0], wf.EscalateTarget(ws))
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "who escalates (default orchestrator)")
	cmd.Flags().StringVar(&why, "why", "", "reason (required)")
	return cmd
}
