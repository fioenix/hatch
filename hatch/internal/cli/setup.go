package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/paths"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
)

// pickableClients are the user-facing client choices, in display order.
// codex/agy wire into HOME config here (their CLIs read MCP only from $HOME);
// claude installs a user-global plugin; kiro is project-scoped (wired by init).
var pickableClients = []struct{ alias, kind, label string }{
	{"cc", "claude", "Claude Code (plugin, user-global)"},
	{"codex", "codex", "Codex (~/.codex/config.toml)"},
	{"agy", "agy", "Antigravity / agy (~/.gemini/config/mcp_config.json)"},
	{"kiro", "kiro", "Kiro (project-scoped — wired per-repo by `hatch init`)"},
}

func newSetupCmd() *cobra.Command {
	var clients []string
	var workflow string
	var yes bool
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "One-time machine onboarding: create the global ~/.hatch and wire your coding-agent CLIs",
		Long: "Run once per machine. Creates the global ~/.hatch workspace (the default in\n" +
			"every repo) and wires the coding-agent CLIs whose MCP config lives in $HOME\n" +
			"(codex, agy) plus the Claude Code plugin. Per-project setup is `hatch init`.\n\n" +
			"With no --client and a terminal, it prompts you to pick. In scripts pass\n" +
			"--client cc,codex,... (and --yes to skip the prompt).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			g := paths.GlobalRoot()
			if g == "" {
				return fmt.Errorf("cannot resolve home dir for ~/.hatch")
			}
			ssot := paths.At(filepath.Dir(g))

			// 1. Global workspace (scaffold once).
			if _, statErr := os.Stat(ssot.Root); statErr == nil {
				fmt.Fprintf(out, "✓ global workspace %s đã tồn tại\n", ssot.Root)
			} else if dryRun {
				fmt.Fprintf(out, "[dry-run] would create %s (workflow=%s)\n", ssot.Root, workflow)
			} else {
				l, written, err := scaffold.Init(scaffold.Options{Dir: filepath.Dir(g), Workflow: workflow})
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Created %s [global] (%d files, workflow=%s)\n", l.Root, len(written), workflow)
			}

			// 2. Decide which clients to wire.
			chosen := splitClients(clients)
			if len(chosen) == 0 {
				if yes || !isInteractive() {
					return fmt.Errorf("no --client given (non-interactive): pass e.g. --client cc,codex")
				}
				chosen = promptClients(cmd)
			}
			if len(chosen) == 0 {
				fmt.Fprintln(out, "Không chọn client nào — bỏ qua wiring. Chạy lại `hatch setup --client …` khi cần.")
				return nil
			}

			// 3. Load the global workspace so client wiring resolves global agent ids.
			//    (Skip under dry-run when it was never created.)
			if dryRun {
				if _, statErr := os.Stat(ssot.Root); statErr != nil {
					fmt.Fprintf(out, "[dry-run] would wire clients: %s\n", strings.Join(chosen, ", "))
					return nil
				}
			}
			ws, err := config.Load(ssot)
			if err != nil {
				return err
			}

			// 4. Wire each chosen client at the right scope.
			for _, alias := range chosen {
				kind, ok := resolveClientKind(alias)
				if !ok {
					fmt.Fprintf(out, "✗ client %q không hợp lệ (cc | codex | agy | kiro)\n", alias)
					continue
				}
				switch kind {
				case "codex":
					// Home-scoped: setupClient writes ~/.codex/config.toml via `codex mcp add`.
					if err := setupClient(cmd, ws, "", alias, dryRun); err != nil {
						return err
					}
					// Lifecycle hook: brief codex on session start from the shared chat.
					if id, ok := agentIDForKind(ws, "codex"); ok {
						home, _ := os.UserHomeDir()
						p := filepath.Join(home, ".codex", "hooks.json")
						cmdStr := "hatch brief --as " + id
						if dryRun {
							fmt.Fprintf(out, "[dry-run] codex: would merge SessionStart hook → %s (`%s`)\n", p, cmdStr)
						} else if added, err := mergeSessionStartHook(p, cmdStr); err != nil {
							fmt.Fprintf(out, "⚠ codex hook: %v\n", err)
						} else if added {
							fmt.Fprintf(out, "✓ codex: SessionStart hook → %s (`%s`)\n", p, cmdStr)
							fmt.Fprintln(out, "  (Codex sẽ hỏi TRUST hook này ở lần chạy tới — duyệt để nó hoạt động)")
						} else {
							fmt.Fprintln(out, "✓ codex: SessionStart hook đã có")
						}
					}
				case "agy":
					// Home-scoped: setupClient writes ~/.gemini/config/mcp_config.json.
					if err := setupClient(cmd, ws, "", alias, dryRun); err != nil {
						return err
					}
				case "claude":
					fmt.Fprintln(out, "claude: cài plugin (skill `hatch-chat` + /hatch) cho Claude Code:")
					fmt.Fprintln(out, "    /plugin marketplace add fioenix/overclaud")
					fmt.Fprintln(out, "    /plugin install hatch@hatch")
				case "kiro":
					fmt.Fprintln(out, "kiro: MCP của kiro là project-scoped — chạy `hatch init --client kiro` trong repo.")
				}
			}

			fmt.Fprintln(out, "\nXong setup máy. Tiếp theo: vào repo của bạn và chạy `hatch init` (mặc định orchestrator = cc).")
			fmt.Fprintln(out, "Kiểm tra: `hatch doctor`.")
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&clients, "client", nil,
		"clients to wire: cc | codex | agy | kiro (repeatable / comma-separated)")
	cmd.Flags().StringVarP(&workflow, "workflow", "w", "scrum",
		"workflow template for the global workspace: "+strings.Join(scaffold.WorkflowTemplates, " | "))
	cmd.Flags().BoolVar(&yes, "yes", false, "non-interactive: never prompt (requires --client)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what setup would do without writing")
	return cmd
}

// mergeSessionStartHook adds a `command`-type SessionStart hook running cmdStr
// into a Claude-Code-style hooks.json (the same schema Codex uses), preserving
// every existing hook. Returns whether it added one (idempotent on cmdStr).
// Writes atomically via a temp file.
func mergeSessionStartHook(path, cmdStr string) (bool, error) {
	root := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &root); err != nil {
			return false, fmt.Errorf("%s không phải JSON hợp lệ: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	groups, _ := hooks["SessionStart"].([]any)
	// Idempotent: bail if cmdStr is already wired under SessionStart.
	for _, g := range groups {
		gm, _ := g.(map[string]any)
		hs, _ := gm["hooks"].([]any)
		for _, h := range hs {
			hm, _ := h.(map[string]any)
			if s, _ := hm["command"].(string); s == cmdStr {
				return false, nil
			}
		}
	}
	groups = append(groups, map[string]any{
		"hooks": []any{map[string]any{"type": "command", "command": cmdStr, "timeout": 10}},
	})
	hooks["SessionStart"] = groups
	root["hooks"] = hooks

	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	tmp := path + ".hatch.tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return false, err
	}
	return true, os.Rename(tmp, path)
}

// isInteractive reports whether stdin is a terminal (so we may prompt).
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// promptClients shows the client list (marking which CLIs are on PATH) and reads
// a comma/space-separated selection. Empty input selects the detected CLIs.
func promptClients(cmd *cobra.Command) []string {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "\nChọn client để wire (vd `1,2` hoặc `cc codex`; Enter = các CLI đã cài):")
	var detected []string
	for i, c := range pickableClients {
		mark := " "
		if bin := defaultCmdForKind[c.kind]; bin != "" {
			if _, err := exec.LookPath(bin); err == nil {
				mark, detected = "✓", append(detected, c.alias)
			}
		}
		fmt.Fprintf(out, "  %d) [%s] %-6s %s\n", i+1, mark, c.alias, c.label)
	}
	fmt.Fprint(out, "> ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return nil // EOF / no real input — don't silently select everything
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return detected // deliberate Enter → the detected CLIs
	}
	return parseClientSelection(line)
}

// parseClientSelection turns a prompt line ("1,2" or "cc codex") into client
// aliases. Single digits map to the pickableClients list (1-based); other tokens
// pass through verbatim (validated later by resolveClientKind).
func parseClientSelection(line string) []string {
	var picked []string
	for _, tok := range strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == ' ' }) {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if len(tok) == 1 && tok[0] >= '1' && tok[0] <= '9' {
			if idx := int(tok[0] - '1'); idx < len(pickableClients) {
				picked = append(picked, pickableClients[idx].alias)
				continue
			}
		}
		picked = append(picked, tok)
	}
	return picked
}
