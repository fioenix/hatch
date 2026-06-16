package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/compile"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

// envForKind names the credential env var each agent kind accepts (besides
// interactive OAuth/login). agy has no documented API-key env (OAuth/keyring).
var envForKind = map[string]string{
	"claude": "ANTHROPIC_API_KEY",
	"codex":  "OPENAI_API_KEY",
	"kiro":   "KIRO_API_KEY",
}

// wiringStatus reports whether an agent kind has its Hatch MCP server and
// session-start hook wired, by inspecting the config each CLI actually reads.
// Claude is wired by its user-global plugin (not a file we can cheaply detect),
// so it reports "plugin"; agy's hook is a Python-SDK plugin (backlog) → "—".
func wiringStatus(kind, repoRoot string) (mcp, hook string) {
	home, _ := os.UserHomeDir()
	has := func(path, needle string) string {
		b, err := os.ReadFile(path)
		if err != nil {
			return "✗"
		}
		if strings.Contains(string(b), needle) {
			return "✓"
		}
		return "✗"
	}
	switch kind {
	case "claude":
		return "plugin", "plugin"
	case "codex":
		return has(filepath.Join(home, ".codex", "config.toml"), "mcp_servers.hatch"),
			has(filepath.Join(home, ".codex", "hooks.json"), "hatch brief")
	case "agy":
		return has(filepath.Join(home, ".gemini", "config", "mcp_config.json"), `"hatch"`),
			has(filepath.Join(home, ".gemini", "config", "hooks.json"), "hatch brief")
	case "kiro":
		return has(filepath.Join(repoRoot, ".kiro", "settings", "mcp.json"), `"hatch"`),
			has(filepath.Join(repoRoot, ".kiro", "cli-agents", "hatch.json"), "hatch brief")
	}
	return "—", "—"
}

// defaultCmdForKind is the executable each kind drives.
var defaultCmdForKind = map[string]string{
	"claude": "claude", "codex": "codex",
	"agy": "agy", "kiro": "kiro-cli", "mock": "hatch-mock",
}

// defaultAuthCheck is a non-mutating, scriptable command (argv) per kind that
// exits 0 when authenticated — verified from each CLI's docs (see
// docs/10-agent-adapters.md). agy has no such command (OAuth/keyring only).
var defaultAuthCheck = map[string][]string{
	"claude": {"auth", "status"},  // exit 0 if logged in, 1 if not (JSON)
	"codex":  {"login", "status"}, // exit 0 if authed
	"kiro":   {"user", "whoami"},  // exit 0 if authed (documented path)
}

// authKinds are agent kinds that require authentication (vs mock/manual/shell).
var authKinds = map[string]bool{"claude": true, "codex": true, "agy": true, "kiro": true}

// authStatus reports how an agent authenticates, WITHOUT touching credential
// files (security): it honours an env key the user set, else runs the agent
// CLI's own auth-check command, else — for CLIs with no scriptable check
// (agy) — reports unknown rather than guessing.
func authStatus(a model.Agent, bin string, cliPresent bool) string {
	ev := envForKind[a.Kind]
	if ev != "" && os.Getenv(ev) != "" {
		return "✓ env " + ev
	}
	check := a.AuthCheck
	if len(check) == 0 {
		check = defaultAuthCheck[a.Kind]
	}
	if len(check) > 0 {
		if !cliPresent {
			return "? (CLI vắng)"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, bin, check...)
		c.Stdin = nil // never block on interactive input
		if err := c.Run(); err == nil {
			return "✓ `" + bin + " " + joinArgs(check) + "`"
		}
		return "✗ chưa login (`" + bin + " login`)"
	}
	if !authKinds[a.Kind] {
		return "—" // mock/manual/shell: no credential needed
	}
	// Needs auth but exposes no scriptable check (OAuth/keyring only).
	if ev != "" {
		return "? OAuth/keyring (hoặc set " + ev + ")"
	}
	return "? OAuth/keyring (login bằng `" + bin + "`)"
}

func joinArgs(a []string) string {
	out := ""
	for i, s := range a {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check readiness: config, compiled freshness, agent CLIs + auth",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			issues := 0
			fmt.Fprintln(out, "Hatch doctor")

			if probs := ws.Validate(); len(probs) > 0 {
				issues += len(probs)
				fmt.Fprintf(out, "✗ config: %d problem(s) (run `hatch validate`)\n", len(probs))
			} else {
				fmt.Fprintln(out, "✓ config valid")
			}

			if reason, _ := compile.StaleReason(ws.Layout, ws.Layout.RepoRoot()); reason != "" {
				issues++
				fmt.Fprintf(out, "✗ compiled stale: %s (run `hatch compile`)\n", reason)
			} else {
				fmt.Fprintln(out, "✓ compiled up to date")
			}

			// Agents: which CLIs are present + how they authenticate. CLIs are
			// NOT mandatory — you only need at least one usable agent.
			fmt.Fprintln(out, "\nAgents (cài cái nào dùng cái nấy — chỉ cần ≥1):")
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "  AGENT\tKIND\tCLI\tAUTH\tMCP\tHOOK")
			present, presentReal := 0, 0
			for _, a := range ws.Registry.Agents {
				bin := a.Cmd
				if bin == "" {
					bin = defaultCmdForKind[a.Kind]
				}
				cliPresent := false
				cli := "n/a"
				if bin != "" {
					if _, err := exec.LookPath(bin); err == nil {
						cli, cliPresent = "✓ "+bin, true
						present++
						if a.Kind != "mock" {
							presentReal++
						}
					} else {
						cli = "✗ " + bin
					}
				}
				mcp, hook := wiringStatus(a.Kind, ws.Layout.RepoRoot())
				fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%s\n", a.ID, a.Kind, cli, authStatus(a, bin, cliPresent), mcp, hook)
			}
			tw.Flush()

			fmt.Fprintln(out)
			fmt.Fprintln(out, "auth: ✓ sẵn sàng · ? không kiểm được (keyring/cấu hình) — login OAuth hoặc env key đều dùng được")
			fmt.Fprintln(out, "MCP/HOOK: ✓ đã wire · ✗ chưa (chạy `hatch setup`/`hatch init`) · plugin: qua Claude plugin · — không áp dụng")
			if present == 0 {
				issues++
				fmt.Fprintln(out, "✗ không có agent CLI nào khả dụng — cài ít nhất 1 (claude/codex/agy/kiro)")
			} else if presentReal == 0 {
				fmt.Fprintln(out, "● chỉ có mock — ổn để test; cài ≥1 agent CLI thật để chạy thật")
			}

			if issues == 0 {
				fmt.Fprintln(out, "✓ ready")
				return nil
			}
			fmt.Fprintf(out, "%d issue(s) — see above.\n", issues)
			return fmt.Errorf("%d readiness issue(s)", issues)
		},
	}
}
