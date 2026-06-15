package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
	"github.com/fioenix/overclaud/hatch/internal/store"
	"github.com/fioenix/overclaud/hatch/internal/wf"
)

func newMobCmd() *cobra.Command {
	var agentsCSV, why string
	var rounds int
	var dryRun, claim bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "mob <ticket>",
		Short: "Mob programming: 3+ agents on one ticket, the driver rotates each round",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			var agents []model.Agent
			for _, id := range splitCSV(agentsCSV) {
				a, ok := ws.Registry.AgentByID(id)
				if !ok {
					return fmt.Errorf("unknown agent %q", id)
				}
				agents = append(agents, a)
			}
			if len(agents) < 2 {
				return fmt.Errorf("mob needs at least 2 agents (use `pair` for 2)")
			}
			b := store.NewBoard(ws.Layout)
			t, ok, err := b.Find(args[0], ws.Workflow.LaneIDs())
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("ticket %s not found", args[0])
			}
			if claim {
				to := claimTarget(ws, t.Lane)
				if to == "" {
					return fmt.Errorf("no claim transition from %q", t.Lane)
				}
				if why == "" {
					why = "mob claim"
				}
				if _, err := engineFor(ws).Move(ws, t.ID, wf.MoveOptions{
					To: to, ByRole: t.Role, Agent: agents[0].ID, Why: why,
				}); err != nil {
					return err
				}
				t, _, _ = b.Find(args[0], ws.Workflow.LaneIDs())
			}
			if rounds < 1 {
				rounds = 1
			}
			thread := "mob-" + t.ID
			bs := bus.New(ws.Layout)
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "mob %s · agents=%s · rounds=%d (driver rotates)\n", t.ID, agentsCSV, rounds)

			for r := 1; r <= rounds; r++ {
				driver := agents[(r-1)%len(agents)]
				// Driver implements.
				raw, _ := bs.Raw(thread)
				fmt.Fprintf(out, "\n# round %d · DRIVER %s\n", r, driver.ID)
				dOut, err := orch(ws).Execute(ws, driver, t.ID,
					orchestrator.BuildPairDriverPrompt(t, thread, raw, "mob"),
					orchestrator.RunOptions{DryRun: dryRun, Timeout: timeout, Stdout: out})
				if err != nil {
					return err
				}
				if dOut.Executed {
					if turn := strings.TrimSpace(dOut.Output); turn != "" {
						bs.Post(bus.Message{Channel: thread, From: driver.ID, To: []string{"*"}, Body: turn})
					}
				}
				// Everyone else navigates.
				ready := 0
				navs := 0
				for _, a := range agents {
					if a.ID == driver.ID {
						continue
					}
					navs++
					raw, _ = bs.Raw(thread)
					fmt.Fprintf(out, "\n# round %d · NAVIGATOR %s\n", r, a.ID)
					nOut, err := orch(ws).Execute(ws, a, t.ID,
						orchestrator.BuildPairNavigatorPrompt(t, thread, raw, driver.ID),
						orchestrator.RunOptions{DryRun: dryRun, Timeout: timeout, Stdout: out})
					if err != nil {
						return err
					}
					if nOut.Executed {
						if turn := strings.TrimSpace(nOut.Output); turn != "" {
							bs.Post(bus.Message{Channel: thread, From: a.ID, To: []string{"*"}, Body: turn})
							if strings.HasPrefix(turn, "READY") {
								ready++
							}
						}
					}
				}
				if navs > 0 && ready > navs/2 {
					fmt.Fprintf(out, "\nđa số navigator READY — kết thúc mob sớm\n")
					break
				}
			}
			fmt.Fprintf(out, "\nmob session in thread %s\n", thread)
			return nil
		},
	}
	cmd.Flags().StringVar(&agentsCSV, "agents", "", "participant agent ids, comma-separated (>=2)")
	cmd.Flags().IntVar(&rounds, "rounds", 2, "rounds; driver rotates each round")
	cmd.Flags().BoolVar(&claim, "claim", false, "claim the ticket to the first agent")
	cmd.Flags().StringVar(&why, "why", "", "reason for the claim ledger entry")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the rotation/turns without running agents")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-turn timeout")
	return cmd
}
