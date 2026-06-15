//go:build hatch_legacy

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
	"github.com/fioenix/overclaud/hatch/internal/store"
	"github.com/fioenix/overclaud/hatch/internal/wf"
)

func newPairCmd() *cobra.Command {
	var driver, navigator, why string
	var rounds int
	var dryRun, claim bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "pair <ticket>",
		Short: "Pair two agents on a ticket: driver implements, navigator reviews each turn",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if driver == "" || navigator == "" {
				return fmt.Errorf("--driver and --navigator are required")
			}
			if driver == navigator {
				return fmt.Errorf("driver and navigator must differ (pairing needs two)")
			}
			drv, ok := ws.Registry.AgentByID(driver)
			if !ok {
				return fmt.Errorf("unknown agent %q", driver)
			}
			nav, ok := ws.Registry.AgentByID(navigator)
			if !ok {
				return fmt.Errorf("unknown agent %q", navigator)
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
					why = "pairing claim"
				}
				if _, err := engineFor(ws).Move(ws, t.ID, wf.MoveOptions{
					To: to, ByRole: t.Role, Agent: drv.ID, Why: why,
				}); err != nil {
					return err
				}
				t, _, _ = b.Find(args[0], ws.Workflow.LaneIDs())
			}
			if rounds < 1 {
				rounds = 1
			}
			thread := "pair-" + t.ID
			bs := bus.New(ws.Layout)
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "pair %s · driver=%s navigator=%s · rounds=%d\n", t.ID, driver, navigator, rounds)

			for r := 1; r <= rounds; r++ {
				// Driver turn.
				raw, _ := bs.Raw(thread)
				fmt.Fprintf(out, "\n# round %d · DRIVER %s\n", r, drv.ID)
				dOut, err := orch(ws).Execute(ws, drv, t.ID,
					orchestrator.BuildPairDriverPrompt(t, thread, raw, nav.ID),
					orchestrator.RunOptions{DryRun: dryRun, Timeout: timeout, Stdout: out})
				if err != nil {
					return err
				}
				if dOut.Executed {
					if turn := strings.TrimSpace(dOut.Output); turn != "" {
						bs.Post(bus.Message{Channel: thread, From: drv.ID, To: []string{nav.ID}, Body: turn})
					}
				}

				// Navigator turn.
				raw, _ = bs.Raw(thread)
				fmt.Fprintf(out, "\n# round %d · NAVIGATOR %s\n", r, nav.ID)
				nOut, err := orch(ws).Execute(ws, nav, t.ID,
					orchestrator.BuildPairNavigatorPrompt(t, thread, raw, drv.ID),
					orchestrator.RunOptions{DryRun: dryRun, Timeout: timeout, Stdout: out})
				if err != nil {
					return err
				}
				if nOut.Executed {
					if turn := strings.TrimSpace(nOut.Output); turn != "" {
						bs.Post(bus.Message{Channel: thread, From: nav.ID, To: []string{drv.ID}, Body: turn})
						if strings.HasPrefix(turn, "READY") {
							fmt.Fprintf(out, "\nnavigator signalled READY — kết thúc pairing sớm\n")
							break
						}
					}
				}
			}
			fmt.Fprintf(out, "\npairing session in thread %s\n", thread)
			return nil
		},
	}
	cmd.Flags().StringVar(&driver, "driver", "", "agent that implements (required)")
	cmd.Flags().StringVar(&navigator, "navigator", "", "agent that reviews each turn (required)")
	cmd.Flags().IntVar(&rounds, "rounds", 3, "max driver/navigator rounds")
	cmd.Flags().BoolVar(&claim, "claim", false, "claim the ticket to the driver first")
	cmd.Flags().StringVar(&why, "why", "", "reason for the claim ledger entry")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the turn structure without running agents")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-turn timeout")
	return cmd
}
