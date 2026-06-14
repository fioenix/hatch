package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/scaffold"
)

func newInitCmd() *cobra.Command {
	var workflow string
	var force bool
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a new .hatch/ workspace from templates",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			l, written, err := scaffold.Init(scaffold.Options{
				Dir:      dir,
				Workflow: workflow,
				Force:    force,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s (%d files, workflow=%s)\n", l.Root, len(written), workflow)
			fmt.Fprintln(cmd.OutOrStdout(), "Next: edit charter.md + registry.yaml, then `hatch compile`.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&workflow, "workflow", "w", "scrum",
		"workflow template: "+strings.Join(scaffold.WorkflowTemplates, " | "))
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .hatch")
	return cmd
}
