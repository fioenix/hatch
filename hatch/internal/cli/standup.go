//go:build hatch_legacy

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/store"
)

func newStandupCmd() *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "standup",
		Short: "Digest recent ledger activity (latest day-files)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			lg := store.NewLedger(ws.Layout)
			files, err := lg.Files()
			if err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no ledger activity yet")
				return nil
			}
			if days < 1 {
				days = 1
			}
			start := 0
			if len(files) > days {
				start = len(files) - days
			}
			out := cmd.OutOrStdout()
			for _, f := range files[start:] {
				raw, err := os.ReadFile(f)
				if err != nil {
					return err
				}
				out.Write(raw)
				fmt.Fprintln(out)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&days, "days", 1, "number of recent ledger day-files to include")
	return cmd
}
