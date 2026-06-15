package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/compile"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

// envForKind names the credential env var each agent kind accepts (besides
// interactive OAuth/login).
var envForKind = map[string]string{
	"claude": "ANTHROPIC_API_KEY",
	"codex":  "OPENAI_API_KEY",
	"gemini": "GEMINI_API_KEY",
	"agy":    "ANTIGRAVITY_API_KEY",
	"kiro":   "KIRO_API_KEY",
}

// defaultCmdForKind is the executable each kind drives.
var defaultCmdForKind = map[string]string{
	"claude": "claude", "codex": "codex", "gemini": "gemini",
	"agy": "agy", "kiro": "kiro-cli", "mock": "hatch-mock",
}

// defaultAuthCheck is a non-mutating, scriptable command (argv) per kind that
// exits 0 when authenticated. Only commands known to be safe + non-interactive
// are listed; for others we don't guess (the user can set `auth_check`).
var defaultAuthCheck = map[string][]string{
	"codex": {"login", "status"},
}

// authStatus reports how an agent authenticates, WITHOUT touching credential
// files: it honours an env key (which the user set), else runs the agent CLI's
// own auth-check command (configurable), else reports "unknown".
func authStatus(a model.Agent, bin string, cliPresent bool) string {
	if ev := envForKind[a.Kind]; ev != "" && os.Getenv(ev) != "" {
		return "✓ env " + ev
	}
	check := a.AuthCheck
	if len(check) == 0 {
		check = defaultAuthCheck[a.Kind]
	}
	if len(check) == 0 {
		if envForKind[a.Kind] == "" {
			return "—"
		}
		return "? set auth_check or run `" + bin + " login`"
	}
	if !cliPresent {
		return "? (CLI vắng)"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, bin, check...)
	c.Stdin = nil // never let the check block on interactive input
	if err := c.Run(); err == nil {
		return "✓ login (`" + bin + " " + joinArgs(check) + "`)"
	}
	return "✗ chưa login (`" + bin + " login`)"
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
			fmt.Fprintln(tw, "  AGENT\tKIND\tCLI\tAUTH")
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
				fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", a.ID, a.Kind, cli, authStatus(a, bin, cliPresent))
			}
			tw.Flush()

			fmt.Fprintln(out)
			fmt.Fprintln(out, "auth: ✓ sẵn sàng · ? không kiểm được (keyring/cấu hình) — login OAuth hoặc env key đều dùng được")
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
