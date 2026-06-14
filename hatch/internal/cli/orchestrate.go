package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
	"github.com/fioenix/overclaud/hatch/internal/presence"
	"github.com/fioenix/overclaud/hatch/internal/store"
	"github.com/fioenix/overclaud/hatch/internal/wf"
)

// pickAgent returns the agent to run a ticket: the explicit id if given, else
// a capacity-aware choice — eligible for the role, present (not paused/offline),
// and least-loaded (under WIP first), like a lead assigning to whoever's free.
func pickAgent(ws *config.Workspace, explicit, role string) (model.Agent, error) {
	if explicit != "" {
		a, ok := ws.Registry.AgentByID(explicit)
		if !ok {
			return model.Agent{}, fmt.Errorf("unknown agent %q", explicit)
		}
		return a, nil
	}
	candidates := ws.Registry.AgentsForRole(role)
	if len(candidates) == 0 {
		return model.Agent{}, fmt.Errorf("no agent in registry holds role %q", role)
	}
	pres := presence.Load(ws.Layout)
	load := wipLoad(ws)

	var free []model.Agent
	for _, a := range candidates {
		if pres.CanTakeWork(a.ID) {
			free = append(free, a)
		}
	}
	if len(free) == 0 {
		return model.Agent{}, fmt.Errorf("no available agent for role %q (all paused/offline)", role)
	}
	sort.SliceStable(free, func(i, j int) bool {
		oi, oj := overWIP(free[i], load), overWIP(free[j], load)
		if oi != oj {
			return !oi // under-WIP first
		}
		if load[free[i].ID] != load[free[j].ID] {
			return load[free[i].ID] < load[free[j].ID] // least loaded
		}
		return free[i].ID < free[j].ID
	})
	return free[0], nil
}

// wipLoad counts in-flight tickets (assignee set, in a non-terminal, non-side
// lane) per agent.
func wipLoad(ws *config.Workspace) map[string]int {
	load := map[string]int{}
	outgoing := map[string]bool{}
	for _, tr := range ws.Workflow.Transitions {
		if tr.From != "*" {
			outgoing[tr.From] = true
		}
	}
	b := store.NewBoard(ws.Layout)
	for _, l := range ws.Workflow.Lanes {
		if l.Side || !outgoing[l.ID] { // skip side + terminal lanes
			continue
		}
		ts, _ := b.ListLane(l.ID)
		for _, t := range ts {
			if t.Assignee != "" {
				load[t.Assignee]++
			}
		}
	}
	return load
}

func overWIP(a model.Agent, load map[string]int) bool {
	return a.WIP > 0 && load[a.ID] >= a.WIP
}

func newRunCmd() *cobra.Command {
	var agentID string
	var dryRun, claim, worktree, noCatchUp bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "run <ticket>",
		Short: "Spawn an agent to work a ticket (headless)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			b := store.NewBoard(ws.Layout)
			t, ok, err := b.Find(args[0], ws.Workflow.LaneIDs())
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("ticket %s not found", args[0])
			}
			agent, err := pickAgent(ws, agentID, t.Role)
			if err != nil {
				return err
			}
			if claim {
				to := claimTarget(ws, t.Lane)
				if to == "" {
					return fmt.Errorf("no claim transition from %q", t.Lane)
				}
				if _, err := wf.Move(ws, b, store.NewLedger(ws.Layout), t.ID, wf.MoveOptions{
					To: to, ByRole: t.Role, Agent: agent.ID, Why: "orchestrator claim",
				}); err != nil {
					return err
				}
				t, _, _ = b.Find(args[0], ws.Workflow.LaneIDs())
			}
			if worktree && !dryRun {
				path, err := orchestrator.AddWorktree(ws.Layout.RepoRoot(), t.ID, t.Branch)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "worktree: %s\n", path)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "→ %s (%s) on %s as %s\n", agent.ID, agent.Kind, t.ID, t.Role)
			out, err := orchestrator.Run(ws, agent, t, t.Role, orchestrator.RunOptions{
				DryRun: dryRun, Timeout: timeout, Stdout: cmd.OutOrStdout(), SkipComms: noCatchUp,
			})
			if err != nil {
				return err
			}
			if out.Executed && out.Err != nil {
				return fmt.Errorf("agent exited with error (code %d)", out.ExitCode)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agentID, "agent", "", "agent id (default: first eligible for the ticket's role)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the invocation without executing")
	cmd.Flags().BoolVar(&claim, "claim", false, "claim the ticket before running")
	cmd.Flags().BoolVar(&worktree, "worktree", false, "run in an isolated git worktree")
	cmd.Flags().BoolVar(&noCatchUp, "no-catch-up", false, "don't prepend inbox + conversation recall")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "kill the agent after this duration (0 = none)")
	return cmd
}

func newPlanCmd() *cobra.Command {
	var agentID string
	var dryRun bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Spawn the Conductor to break work into tickets",
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
			_, err = orchestrator.Execute(ws, agent, "-", orchestrator.BuildPlanPrompt(), orchestrator.RunOptions{
				DryRun: dryRun, Timeout: timeout, Stdout: cmd.OutOrStdout(),
			})
			return err
		},
	}
	cmd.Flags().StringVar(&agentID, "agent", "", "agent id (default: first with role conductor)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the invocation without executing")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "kill the agent after this duration (0 = none)")
	return cmd
}

func newWatchCmd() *cobra.Command {
	var dryRun bool
	var max int
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Assign and run claimable backlog tickets (one pass, respects WIP)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			b := store.NewBoard(ws.Layout)
			lane := firstLane(ws)
			tickets, err := b.ListLane(lane)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			ran := 0
			for _, t := range tickets {
				if max > 0 && ran >= max {
					break
				}
				if t.Role == "" {
					fmt.Fprintf(out, "skip %s: no role\n", t.ID)
					continue
				}
				agent, err := pickAgent(ws, "", t.Role)
				if err != nil {
					fmt.Fprintf(out, "skip %s: %v\n", t.ID, err)
					continue
				}
				fmt.Fprintf(out, "→ %s → %s (%s)\n", t.ID, agent.ID, agent.Kind)
				if dryRun {
					ran++
					continue
				}
				to := claimTarget(ws, t.Lane)
				if _, err := wf.Move(ws, b, store.NewLedger(ws.Layout), t.ID, wf.MoveOptions{
					To: to, ByRole: t.Role, Agent: agent.ID, Why: "watch claim",
				}); err != nil {
					fmt.Fprintf(out, "  claim failed: %v\n", err)
					continue
				}
				t2, _, _ := b.Find(t.ID, ws.Workflow.LaneIDs())
				if _, err := orchestrator.Run(ws, agent, t2, t2.Role, orchestrator.RunOptions{Stdout: out}); err != nil {
					fmt.Fprintf(out, "  run failed: %v\n", err)
				}
				ran++
			}
			fmt.Fprintf(out, "watch: %d ticket(s) dispatched\n", ran)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show assignments without claiming/running")
	cmd.Flags().IntVar(&max, "max", 0, "max tickets to dispatch this pass (0 = all)")
	return cmd
}
