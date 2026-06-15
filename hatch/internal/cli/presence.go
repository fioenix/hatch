//go:build hatch_legacy

package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/presence"
)

func newPresenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "presence",
		Short: "Show agent availability + current load",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			board := presence.Load(ws.Layout)
			load := wipLoad(ws)
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "AGENT\tSTATUS\tWIP\tROLES\tNOTE")
			for _, a := range ws.Registry.Agents {
				st := board[a.ID]
				status := board.StatusOf(a.ID)
				wip := fmt.Sprintf("%d", load[a.ID])
				if a.WIP > 0 {
					wip = fmt.Sprintf("%d/%d", load[a.ID], a.WIP)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%v\t%s\n", a.ID, status, wip, a.Roles, st.Note)
			}
			tw.Flush()
			return nil
		},
	}
	cmd.AddCommand(newPresenceSetCmd())
	return cmd
}

func newPresenceSetCmd() *cobra.Command {
	var status, note string
	cmd := &cobra.Command{
		Use:   "set <agent>",
		Short: "Set an agent's availability (available|busy|paused|offline)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			switch status {
			case presence.Available, presence.Busy, presence.Paused, presence.Offline:
			default:
				return fmt.Errorf("status must be available|busy|paused|offline")
			}
			if _, ok := ws.Registry.AgentByID(args[0]); !ok {
				return fmt.Errorf("unknown agent %q", args[0])
			}
			board := presence.Load(ws.Layout)
			board.Set(args[0], status, note)
			if err := board.Save(ws.Layout); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s → %s\n", args[0], status)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "available|busy|paused|offline (required)")
	cmd.Flags().StringVar(&note, "note", "", "optional note (e.g. PTO until Fri)")
	return cmd
}
