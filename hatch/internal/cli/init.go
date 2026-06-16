package cli

import (
	"fmt"
	"os"
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
	var local bool
	var clients []string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create the global ~/.hatch workspace (or a local one with --local); optionally wire a client",
		Long: "By default `hatch init` creates the user-level workspace at ~/.hatch (like ~/.claude),\n" +
			"used as the default in every repo. Use --local to create a project .hatch in the\n" +
			"current repo that OVERRIDES the global one. Pass [dir] to target an explicit directory.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			// Where the .hatch SSOT lives: explicit dir > --local (cwd) > global (~).
			scaffoldDir := ""
			switch {
			case len(args) == 1:
				scaffoldDir = args[0]
			case local:
				scaffoldDir = "."
			default:
				g := paths.GlobalRoot()
				if g == "" {
					return fmt.Errorf("cannot resolve home dir for ~/.hatch; use --local or pass a dir")
				}
				scaffoldDir = filepath.Dir(g) // parent of ~/.hatch
			}
			absScaffold, _ := filepath.Abs(scaffoldDir)
			ssot := paths.At(absScaffold)
			scope := "global (~/.hatch)"
			if local || len(args) == 1 {
				scope = "local override"
			}

			_, statErr := os.Stat(ssot.Root)
			exists := statErr == nil
			// Scaffold unless it already exists and we're only wiring a client.
			switch {
			case exists && len(clients) > 0 && !force:
				fmt.Fprintf(out, "Workspace %s đã tồn tại — bỏ qua scaffold, chỉ set up client.\n", ssot.Root)
			case dryRun:
				// --dry-run must not touch disk: preview the scaffold instead.
				fmt.Fprintf(out, "[dry-run] would create %s [%s] (workflow=%s)\n", ssot.Root, scope, workflow)
			default:
				l, written, err := scaffold.Init(scaffold.Options{Dir: absScaffold, Workflow: workflow, Force: force})
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Created %s [%s] (%d files, workflow=%s)\n", l.Root, scope, len(written), workflow)
				if len(clients) == 0 {
					fmt.Fprintln(out, "Next: edit charter.md + registry.yaml, then `hatch compile` (hoặc `hatch init --client cc`).")
				}
			}

			if len(clients) == 0 {
				return nil
			}
			if dryRun && !exists {
				fmt.Fprintln(out, "[dry-run] client setup preview cần workspace có sẵn — chạy thật để scaffold trước.")
				return nil
			}

			// Load that workspace; compiled outputs + client config go to the
			// current repo (cwd), even when the SSOT is the global ~/.hatch.
			ws, err := config.Load(ssot)
			if err != nil {
				return err
			}
			ws.OutputRoot = cwd
			if !dryRun {
				if _, _, err := compile.Run(ws); err != nil {
					return fmt.Errorf("compile: %w", err)
				}
				fmt.Fprintf(out, "Compiled surfaces + MCP registration vào %s.\n", cwd)
			}
			for _, c := range splitClients(clients) {
				if err := setupClient(cmd, ws, cwd, c, dryRun); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&workflow, "workflow", "w", "scrum",
		"workflow template: "+strings.Join(scaffold.WorkflowTemplates, " | "))
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .hatch")
	cmd.Flags().BoolVar(&local, "local", false, "create a project .hatch in the current repo (overrides ~/.hatch)")
	cmd.Flags().StringSliceVar(&clients, "client", nil,
		"set up MCP for a client and wire it into the current repo: cc | codex | agy | kiro (repeatable / comma-separated)")
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
