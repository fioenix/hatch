package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fioenix/hatch/internal/compile"
)

func newCompileCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile the SSOT into per-agent instruction files",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if check {
				reason, err := compile.StaleReason(ws.Layout, ws.Layout.RepoRoot())
				if err != nil {
					return err
				}
				if reason != "" {
					return fmt.Errorf("compiled files are stale: %s\nrun `hatch compile`", reason)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "compiled files are up to date")
				return nil
			}
			res, warnings, err := compile.Run(ws)
			if err != nil {
				return err
			}
			repoRoot := ws.Layout.RepoRoot()
			for _, w := range warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: "+w)
			}
			for _, out := range res.Written {
				rel, _ := filepath.Rel(repoRoot, out)
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s\n", rel)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compiled %d surface(s).\n", len(res.Bundles))
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "verify compiled files are up to date (exit non-zero if stale)")
	return cmd
}
