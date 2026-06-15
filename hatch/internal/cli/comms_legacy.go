//go:build hatch_legacy

// ask/convene drive agents synchronously (the orchestrator relays turns). In
// the embedded-harness model agents converse asynchronously through the Hatch
// MCP server instead, so these are archived behind the `hatch_legacy` build
// tag. Build with `-tags hatch_legacy` to restore them.
package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/decide"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
)

// roleOf returns an agent's primary role for framing conversation prompts.
func roleOf(a model.Agent) string {
	if len(a.Roles) > 0 {
		return a.Roles[0]
	}
	return "teammate"
}

func newAskCmd() *cobra.Command {
	var from, to, thread string
	var dryRun bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Ask another agent a question and get a reply (synchronous relay)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required")
			}
			target, ok := ws.Registry.AgentByID(to)
			if !ok {
				return fmt.Errorf("unknown agent %q", to)
			}
			if thread == "" {
				thread = "ask-" + time.Now().Format("0102-150405")
			}
			question := strings.Join(args, " ")
			bs := bus.New(ws.Layout)
			if _, err := bs.Post(bus.Message{Channel: thread, From: from, To: []string{to}, Type: bus.TypeAsk, Body: question}); err != nil {
				return err
			}
			raw, _ := bs.Raw(thread)
			prompt := orchestrator.BuildConsultPrompt(from, roleOf(target), thread, raw, question)
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s → %s (thread %s)\n", from, to, thread)
			outcome, err := orch(ws).Execute(ws, target, thread, prompt, orchestrator.RunOptions{
				DryRun: dryRun, Timeout: timeout, Stdout: out,
			})
			if err != nil {
				return err
			}
			if !outcome.Executed {
				return nil // dry-run or manual handoff; nothing to record yet
			}
			reply := strings.TrimSpace(outcome.Output)
			if reply == "" {
				reply = "(no reply captured)"
			}
			if _, err := bs.Post(bus.Message{Channel: thread, From: to, To: []string{from}, Type: bus.TypeReply, Body: reply}); err != nil {
				return err
			}
			fmt.Fprintf(out, "\n--- reply from %s recorded to thread %s ---\n", to, thread)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "asking agent (or human:<name>)")
	cmd.Flags().StringVar(&to, "to", "", "agent to ask (must be in registry)")
	cmd.Flags().StringVar(&thread, "thread", "", "thread id (default: generated)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the relay invocation without running")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "kill the agent after this duration")
	return cmd
}

func newConveneCmd() *cobra.Command {
	var thread, topic, agentsCSV, chair, decider string
	var rounds int
	var dryRun bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "convene",
		Short: "Run a multi-agent meeting: agents take turns on a topic (human simulation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if topic == "" || agentsCSV == "" {
				return fmt.Errorf("--topic and --agents are required")
			}
			if thread == "" {
				thread = "meet-" + time.Now().Format("0102-150405")
			}
			if rounds < 1 {
				rounds = 1
			}
			if chair == "" {
				chair = "human:facilitator"
			}
			var participants []model.Agent
			for _, id := range splitCSV(agentsCSV) {
				a, ok := ws.Registry.AgentByID(id)
				if !ok {
					return fmt.Errorf("unknown agent %q", id)
				}
				participants = append(participants, a)
			}
			bs := bus.New(ws.Layout)
			out := cmd.OutOrStdout()
			// Kickoff.
			if _, err := bs.Post(bus.Message{Channel: thread, From: chair, To: []string{"*"}, Type: bus.TypeMsg,
				Body: "Họp: " + topic}); err != nil {
				return err
			}
			fmt.Fprintf(out, "convene thread=%s rounds=%d agents=%s\n", thread, rounds, agentsCSV)
			decided := false
			recordDecision := func(by, turn string) {
				body := strings.TrimSpace(strings.TrimPrefix(turn, "DECISION:"))
				if e, err := decide.Record(ws, thread, topic, by, body); err == nil {
					decided = true
					fmt.Fprintf(out, "  ↳ recorded %s in kb/decisions/\n", e.ID)
				}
			}
			for r := 1; r <= rounds && !decided; r++ {
				for _, a := range participants {
					raw, _ := bs.Raw(thread)
					prompt := orchestrator.BuildMeetingPrompt(roleOf(a), thread, topic, raw, r, rounds)
					fmt.Fprintf(out, "\n# round %d · %s (%s)\n", r, a.ID, roleOf(a))
					outcome, err := orch(ws).Execute(ws, a, thread, prompt, orchestrator.RunOptions{
						DryRun: dryRun, Timeout: timeout, Stdout: out,
					})
					if err != nil {
						return err
					}
					if !outcome.Executed {
						continue
					}
					turn := strings.TrimSpace(outcome.Output)
					if turn == "" {
						continue
					}
					tt := turnType(turn)
					if _, err := bs.Post(bus.Message{Channel: thread, From: a.ID, To: []string{"*"}, Type: tt, Body: turn}); err != nil {
						return err
					}
					// A meeting decision becomes a durable ADR in the KB.
					if tt == bus.TypeDecision {
						recordDecision(a.ID, turn)
						break
					}
				}
			}
			// Tie-breaker: no consensus after all rounds → the decider settles it.
			if !decided && decider != "" {
				d, ok := ws.Registry.AgentByID(decider)
				if !ok {
					return fmt.Errorf("unknown decider %q", decider)
				}
				raw, _ := bs.Raw(thread)
				fmt.Fprintf(out, "\n# tie-break · %s (%s)\n", d.ID, roleOf(d))
				outcome, err := orch(ws).Execute(ws, d, thread,
					orchestrator.BuildTieBreakPrompt(roleOf(d), thread, topic, raw),
					orchestrator.RunOptions{DryRun: dryRun, Timeout: timeout, Stdout: out})
				if err != nil {
					return err
				}
				if outcome.Executed {
					turn := strings.TrimSpace(outcome.Output)
					bs.Post(bus.Message{Channel: thread, From: d.ID, To: []string{"*"}, Type: bus.TypeDecision, Body: turn})
					recordDecision(d.ID, turn)
				}
			}
			if !decided && decider == "" && !dryRun {
				fmt.Fprintln(out, "\n(không đạt đồng thuận; chạy lại với --decider <agent> để phân xử)")
			}
			fmt.Fprintf(out, "\nmeeting recorded in thread %s\n", thread)
			return nil
		},
	}
	cmd.Flags().StringVar(&decider, "decider", "", "agent that breaks a tie if no DECISION is reached")
	cmd.Flags().StringVar(&thread, "thread", "", "thread id (default: generated)")
	cmd.Flags().StringVar(&topic, "topic", "", "meeting topic (required)")
	cmd.Flags().StringVar(&agentsCSV, "agents", "", "participant agent ids, comma-separated (required)")
	cmd.Flags().StringVar(&chair, "chair", "", "who convenes (default human:facilitator)")
	cmd.Flags().IntVar(&rounds, "rounds", 1, "number of discussion rounds")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show turn order/invocations without running agents")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-turn timeout")
	return cmd
}

func turnType(body string) string {
	if strings.HasPrefix(strings.TrimSpace(body), "DECISION:") {
		return bus.TypeDecision
	}
	return bus.TypeMsg
}
