package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/compile"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/paths"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
)

func newInitCmd() *cobra.Command {
	var workflow string
	var force bool
	var clients []string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a new .hatch/ workspace; optionally set up a client (--client cc|codex|agy|kiro)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			out := cmd.OutOrStdout()

			// Scaffold unless a workspace already exists and we're only wiring a
			// client into it (so `hatch init --client codex` works in-place).
			existing := false
			if _, err := paths.Find(dir); err == nil {
				existing = true
			}
			if existing && len(clients) > 0 && !force {
				fmt.Fprintf(out, "Workspace .hatch đã tồn tại — bỏ qua scaffold, chỉ set up client.\n")
			} else {
				l, written, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: workflow, Force: force})
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Created %s (%d files, workflow=%s)\n", l.Root, len(written), workflow)
				if len(clients) == 0 {
					fmt.Fprintln(out, "Next: edit charter.md + registry.yaml, then `hatch compile`.")
				}
			}

			if len(clients) == 0 {
				return nil
			}

			// Load the workspace and compile so each client has its instruction
			// surface + base MCP registration, then wire the requested clients.
			absDir, _ := filepath.Abs(dir)
			ws, err := config.Load(paths.At(absDir))
			if err != nil {
				return err
			}
			if !dryRun {
				if _, _, err := compile.Run(ws); err != nil {
					return fmt.Errorf("compile: %w", err)
				}
				fmt.Fprintln(out, "Compiled surfaces + MCP registration.")
			}
			for _, c := range splitClients(clients) {
				if err := setupClient(cmd, ws, absDir, c, dryRun); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&workflow, "workflow", "w", "scrum",
		"workflow template: "+strings.Join(scaffold.WorkflowTemplates, " | "))
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .hatch")
	cmd.Flags().StringSliceVar(&clients, "client", nil,
		"set up MCP for a client and exit-wire it: cc | codex | agy | kiro (repeatable / comma-separated)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what --client setup would do without writing")
	return cmd
}

// splitClients flattens comma-separated values inside the repeatable flag.
func splitClients(in []string) []string {
	var out []string
	for _, v := range in {
		for _, p := range strings.Split(v, ",") {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}
