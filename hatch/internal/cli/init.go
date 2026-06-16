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
	var global bool
	var client string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Set up Hatch in the current repo: create a local .hatch, pick the orchestrator, compile",
		Long: "Run inside a project repo. Creates a local .hatch (overriding the global\n" +
			"~/.hatch from `hatch setup`), picks one client as the orchestrator\n" +
			"(--client, default cc), compiles the surfaces, and wires the project-scoped\n" +
			"agents (claude .mcp.json, kiro .kiro/) so the squad reaches the chat.\n\n" +
			"Use --global to target ~/.hatch instead, or pass [dir] for an explicit path.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			// Where the .hatch SSOT lives: explicit dir > --global (~) > local (cwd).
			scaffoldDir := "."
			scope := "local"
			switch {
			case len(args) == 1:
				scaffoldDir = args[0]
			case global:
				g := paths.GlobalRoot()
				if g == "" {
					return fmt.Errorf("cannot resolve home dir for ~/.hatch; pass a dir")
				}
				scaffoldDir = filepath.Dir(g) // parent of ~/.hatch
				scope = "global (~/.hatch)"
			}
			absScaffold, _ := filepath.Abs(scaffoldDir)
			ssot := paths.At(absScaffold)

			_, statErr := os.Stat(ssot.Root)
			exists := statErr == nil
			switch {
			case dryRun:
				if exists {
					fmt.Fprintf(out, "[dry-run] workspace %s đã tồn tại — would skip scaffold\n", ssot.Root)
				} else {
					fmt.Fprintf(out, "[dry-run] would create %s [%s] (workflow=%s)\n", ssot.Root, scope, workflow)
				}
				fmt.Fprintf(out, "[dry-run] orchestrator=%s, then compile + wire project-scoped agents (claude/kiro).\n", client)
				return nil
			case exists && !force:
				fmt.Fprintf(out, "Workspace %s đã tồn tại — bỏ qua scaffold.\n", ssot.Root)
			default:
				l, written, err := scaffold.Init(scaffold.Options{Dir: absScaffold, Workflow: workflow, Force: force})
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Created %s [%s] (%d files, workflow=%s)\n", l.Root, scope, len(written), workflow)
			}

			// Load workspace; compiled surfaces + project client configs land in cwd.
			ws, err := config.Load(ssot)
			if err != nil {
				return err
			}
			ws.OutputRoot = cwd

			// Pick the orchestrator (the conductor seat) from --client.
			kind, ok := resolveClientKind(client)
			if !ok {
				return fmt.Errorf("unknown --client %q (use: cc | codex | agy | kiro)", client)
			}
			leadID, ok := agentIDForKind(ws, kind)
			if !ok {
				return fmt.Errorf("no %s-kind agent in registry.yaml — add one, then re-run", kind)
			}
			if ws.Registry.Orchestrator != leadID {
				if err := setRegistryOrchestrator(ssot.Registry(), leadID); err != nil {
					return fmt.Errorf("set orchestrator: %w", err)
				}
				ws.Registry.Orchestrator = leadID
			}
			fmt.Fprintf(out, "Orchestrator: %s (%s) — orchestrator block vào surface của nó.\n", leadID, kind)

			if _, _, err := compile.Run(ws); err != nil {
				return fmt.Errorf("compile: %w", err)
			}
			// compile writes the per-agent MCP registration: kiro's
			// .kiro/settings/mcp.json (its only wiring point) plus paste snippets for
			// codex/agy under .hatch/mcp/. claude/codex/agy were wired machine-wide by
			// `hatch setup` (plugin / ~/.codex / ~/.gemini), so init adds nothing for
			// them here.
			fmt.Fprintf(out, "Compiled surfaces + MCP registration vào %s.\n", cwd)

			// Keep the local .hatch out of git by default: it holds per-developer
			// workspace state (board/ledger/kb). The compiled surfaces
			// (CLAUDE.md/AGENTS.md/GEMINI.md) stay tracked — they are the agents'
			// committed project instructions.
			if !global {
				if added, err := ensureGitignore(cwd, "/.hatch/"); err != nil {
					fmt.Fprintf(out, "⚠ không cập nhật được .gitignore: %v\n", err)
				} else if added {
					fmt.Fprintln(out, "✓ .gitignore += /.hatch/ (workspace local — không commit)")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&workflow, "workflow", "w", "scrum",
		"workflow template: "+strings.Join(scaffold.WorkflowTemplates, " | "))
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .hatch")
	cmd.Flags().BoolVar(&local, "local", true, "create the .hatch in the current repo (default; overrides ~/.hatch)")
	cmd.Flags().BoolVar(&global, "global", false, "target the global ~/.hatch instead of a local .hatch")
	cmd.Flags().StringVar(&client, "client", "cc", "client to seat as orchestrator: cc | codex | agy | kiro")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what init would do without writing")
	_ = local // --local is the default; kept for backward compatibility
	return cmd
}

// ensureGitignore appends pattern to repoRoot/.gitignore unless it is already
// listed. Returns whether it added the line. Creates the file if absent.
func ensureGitignore(repoRoot, pattern string) (bool, error) {
	path := filepath.Join(repoRoot, ".gitignore")
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	for _, ln := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(ln) == pattern {
			return false, nil
		}
	}
	var b strings.Builder
	b.Write(raw)
	if len(raw) > 0 && !strings.HasSuffix(string(raw), "\n") {
		b.WriteString("\n")
	}
	b.WriteString(pattern + "\n")
	return true, os.WriteFile(path, []byte(b.String()), 0o644)
}

// setRegistryOrchestrator writes/updates the top-level `orchestrator:` key in
// registry.yaml via a targeted text edit, preserving all comments. It replaces an
// existing line or inserts one right after the `version:` line.
func setRegistryOrchestrator(path, agentID string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	newLine := "orchestrator: " + agentID
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "orchestrator:") {
			lines[i] = newLine
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
		}
	}
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "version:") {
			rest := append([]string{newLine}, lines[i+1:]...)
			lines = append(lines[:i+1], rest...)
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
		}
	}
	// No version line — prepend.
	return os.WriteFile(path, []byte(newLine+"\n"+string(raw)), 0o644)
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
