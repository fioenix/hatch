package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/ceremony"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
)

func newCeremonyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ceremony",
		Aliases: []string{"cer"},
		Short:   "Run squad rituals: standup, retro, planning",
	}
	cmd.AddCommand(newStandupCeremonyCmd(), newRetroCmd(), newPlanningCmd(), newDemoCmd(), newGroomingCmd())
	return cmd
}

func newDemoCmd() *cobra.Command {
	var post bool
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Showcase completed work (sprint review); posts to #demo",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			report, _, err := ceremony.Demo(ws)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), report)
			if post {
				if _, err := bus.New(ws.Layout).Post(bus.Message{
					Channel: "#demo", From: ceremonyChair(ws, "demo"), To: []string{"*"}, Body: report,
				}); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "\n(posted to #demo)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&post, "post", true, "post the showcase to #demo")
	return cmd
}

func newGroomingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grooming",
		Short: "Flag under-specified backlog tickets (missing role/priority/acceptance)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			report, items, err := ceremony.Grooming(ws)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), report)
			if len(items) > 0 {
				return fmt.Errorf("%d backlog ticket(s) need refinement", len(items))
			}
			return nil
		},
	}
	return cmd
}

func newStandupCeremonyCmd() *cobra.Command {
	var days int
	var post bool
	cmd := &cobra.Command{
		Use:   "standup",
		Short: "Per-agent digest of recent activity + blockers (posts to #standup)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			rep, err := ceremony.Standup(ws, days)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprint(out, rep.Markdown)
			if post {
				chair := ceremonyChair(ws, "standup-digest")
				if _, err := bus.New(ws.Layout).Post(bus.Message{
					Channel: "#standup", From: chair, To: []string{"*"}, Body: rep.Markdown,
				}); err != nil {
					return err
				}
				fmt.Fprintln(out, "\n(posted to #standup)")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&days, "days", 1, "days of ledger history to digest")
	cmd.Flags().BoolVar(&post, "post", true, "post the digest to #standup")
	return cmd
}

func newRetroCmd() *cobra.Command {
	var write bool
	cmd := &cobra.Command{
		Use:   "retro",
		Short: "Cycle summary + KB→SSOT promotion candidates",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			rep, err := ceremony.Retro(ws)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprint(out, rep.Markdown)
			if write {
				name := "retro-" + time.Now().Format("2006-01-02") + ".md"
				p := filepath.Join(ws.Layout.Ledger(), name)
				if err := os.WriteFile(p, []byte(rep.Markdown), 0o644); err != nil {
					return err
				}
				fmt.Fprintf(out, "\n(wrote %s)\n", p)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "save the retro summary under ledger/")
	return cmd
}

func newPlanningCmd() *cobra.Command {
	var agentID string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "planning",
		Short: "Spawn the Conductor to plan the next cycle",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			agent, err := pickAgent(ws, agentID, "conductor")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "→ planning with %s (%s)\n", agent.ID, agent.Kind)
			_, err = orch(ws).Execute(ws, agent, "-", orchestrator.BuildPlanPrompt(), orchestrator.RunOptions{
				DryRun: dryRun, Stdout: cmd.OutOrStdout(),
			})
			return err
		},
	}
	cmd.Flags().StringVar(&agentID, "agent", "", "agent id (default: first with role conductor)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the invocation without running")
	return cmd
}

// ceremonyChair resolves who chairs a ceremony from workflow.yaml, falling back
// to a human facilitator.
func ceremonyChair(ws *config.Workspace, name string) string {
	if c, ok := ws.Workflow.Ceremonies[name]; ok && c.By != "" {
		return c.By
	}
	return "human:facilitator"
}
