package cli

import (
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/compile"
)

// envForKind names the credential env var each agent kind expects (best-effort).
var envForKind = map[string]string{
	"claude": "ANTHROPIC_API_KEY",
	"codex":  "OPENAI_API_KEY",
	"gemini": "GEMINI_API_KEY",
	"kiro":   "KIRO_API_KEY",
}

// defaultCmdForKind is the executable each kind drives.
var defaultCmdForKind = map[string]string{
	"claude": "claude", "codex": "codex", "gemini": "gemini", "kiro": "kiro-cli", "mock": "hatch-mock",
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check readiness: config, compiled freshness, agent CLIs + credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			issues := 0
			fmt.Fprintln(out, "Hatch doctor")

			// Config validity.
			if probs := ws.Validate(); len(probs) > 0 {
				issues += len(probs)
				fmt.Fprintf(out, "✗ config: %d problem(s) (run `hatch validate`)\n", len(probs))
			} else {
				fmt.Fprintln(out, "✓ config valid")
			}

			// Compiled freshness.
			if reason, _ := compile.StaleReason(ws.Layout, ws.Layout.RepoRoot()); reason != "" {
				issues++
				fmt.Fprintf(out, "✗ compiled stale: %s (run `hatch compile`)\n", reason)
			} else {
				fmt.Fprintln(out, "✓ compiled up to date")
			}

			// Agents: CLI on PATH + credential env.
			fmt.Fprintln(out, "\nAgents:")
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "  AGENT\tKIND\tCLI\tCREDENTIAL")
			for _, a := range ws.Registry.Agents {
				bin := a.Cmd
				if bin == "" {
					bin = defaultCmdForKind[a.Kind]
				}
				cli := "n/a"
				if bin != "" {
					if _, err := exec.LookPath(bin); err == nil {
						cli = "✓ " + bin
					} else {
						cli = "✗ " + bin + " not on PATH"
						if a.Kind != "mock" && a.Kind != "manual" {
							issues++
						}
					}
				}
				cred := "—"
				if ev := envForKind[a.Kind]; ev != "" {
					if os.Getenv(ev) != "" {
						cred = "✓ " + ev
					} else {
						cred = "✗ " + ev + " unset"
					}
				}
				fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", a.ID, a.Kind, cli, cred)
			}
			tw.Flush()

			fmt.Fprintln(out)
			if issues == 0 {
				fmt.Fprintln(out, "✓ ready")
				return nil
			}
			fmt.Fprintf(out, "%d issue(s) — see above. (mock agents need no CLI/credential.)\n", issues)
			return fmt.Errorf("%d readiness issue(s)", issues)
		},
	}
}
