package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/oncall"
)

func newOncallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oncall",
		Short: "Show who is on call (first responder for escalations)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			r := oncall.Load(ws.Layout)
			out := cmd.OutOrStdout()
			if r.Now() == "" {
				fmt.Fprintln(out, "no on-call rotation set (use `hatch oncall set --rotation a,b,c`)")
				return nil
			}
			fmt.Fprintf(out, "on-call: %s\nrotation: %v\n", r.Now(), r.Order)
			return nil
		},
	}
	cmd.AddCommand(newOncallSetCmd(), newOncallRotateCmd())
	return cmd
}

func newOncallSetCmd() *cobra.Command {
	var rotationCSV string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Define the on-call rotation (ordered agent ids)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			order := splitCSV(rotationCSV)
			if len(order) == 0 {
				return fmt.Errorf("--rotation is required (comma-separated agent ids)")
			}
			for _, id := range order {
				if _, ok := ws.Registry.AgentByID(id); !ok {
					return fmt.Errorf("unknown agent %q", id)
				}
			}
			r := oncall.Rotation{Order: order, Current: 0}
			if err := r.Save(ws.Layout); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "rotation set; on-call: %s\n", r.Now())
			return nil
		},
	}
	cmd.Flags().StringVar(&rotationCSV, "rotation", "", "ordered agent ids, comma-separated (required)")
	return cmd
}

func newOncallRotateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Advance the pager to the next agent (announces in #oncall)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			r := oncall.Load(ws.Layout)
			prev := r.Now()
			next := r.Rotate()
			if next == "" {
				return fmt.Errorf("no rotation set")
			}
			if err := r.Save(ws.Layout); err != nil {
				return err
			}
			_, _ = bus.New(ws.Layout).Post(bus.Message{
				Channel: "#oncall", From: "human:facilitator", To: []string{next},
				Type: bus.TypeMsg, Body: fmt.Sprintf("@%s bạn nhận pager on-call (bàn giao từ %s)", next, prev),
			})
			fmt.Fprintf(cmd.OutOrStdout(), "on-call: %s → %s\n", prev, next)
			return nil
		},
	}
	return cmd
}
