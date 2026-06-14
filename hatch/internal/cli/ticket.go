package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
	"github.com/fioenix/overclaud/hatch/internal/wf"
)

func newTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ticket",
		Aliases: []string{"t"},
		Short:   "Create and move tickets on the board",
	}
	cmd.AddCommand(newTicketNewCmd(), newTicketClaimCmd(), newTicketMoveCmd(), newTicketShowCmd(), newTicketExtdepCmd())
	return cmd
}

func newTicketExtdepCmd() *cobra.Command {
	var add, owner, eta, resolve string
	cmd := &cobra.Command{
		Use:   "extdep <id>",
		Short: "Manage external/cross-team blockers on a ticket",
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
			out := cmd.OutOrStdout()
			switch {
			case add != "":
				t.BlockedByExternal = append(t.BlockedByExternal, model.ExternalBlocker{
					What: add, Owner: owner, ETA: eta, Status: "waiting",
				})
			case resolve != "":
				found := false
				for i := range t.BlockedByExternal {
					if t.BlockedByExternal[i].What == resolve {
						t.BlockedByExternal[i].Status = "received"
						found = true
					}
				}
				if !found {
					return fmt.Errorf("no external blocker matching %q", resolve)
				}
			default:
				if len(t.BlockedByExternal) == 0 {
					fmt.Fprintln(out, "no external blockers")
					return nil
				}
				for _, e := range t.BlockedByExternal {
					fmt.Fprintf(out, "- [%s] %s (owner %s, eta %s)\n", e.Status, e.What, e.Owner, e.ETA)
				}
				return nil
			}
			if _, err := b.Write(t); err != nil {
				return err
			}
			_ = store.NewLedger(ws.Layout).Append(model.Entry{
				Agent: "human:operator", Ticket: t.ID, Action: model.ActNote,
				Why: "external dependency updated",
			})
			fmt.Fprintf(out, "updated external blockers on %s\n", t.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&add, "add", "", "add an external blocker with this description")
	cmd.Flags().StringVar(&owner, "owner", "", "blocker owner (e.g. human:vendor)")
	cmd.Flags().StringVar(&eta, "eta", "", "expected resolution date")
	cmd.Flags().StringVar(&resolve, "resolve", "", "mark a blocker (by description) received")
	return cmd
}

// firstLane returns the workflow's entry lane (first non-side lane).
func firstLane(ws *config.Workspace) string {
	for _, l := range ws.Workflow.Lanes {
		if !l.Side {
			return l.ID
		}
	}
	return ws.Workflow.Lanes[0].ID
}

func newTicketNewCmd() *cobra.Command {
	var title, role, priority, epic string
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a ticket in the entry lane",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if role != "" {
				if _, ok := ws.Registry.RoleByID(role); !ok {
					return fmt.Errorf("unknown role %q", role)
				}
			}
			b := store.NewBoard(ws.Layout)
			id, err := b.NextID(ws.Workflow.LaneIDs())
			if err != nil {
				return err
			}
			lane := firstLane(ws)
			now := time.Now().Format(time.RFC3339)
			t := model.Ticket{
				ID:       id,
				Title:    title,
				Status:   lane,
				Role:     role,
				Priority: priority,
				Epic:     epic,
				Created:  now,
				Updated:  now,
				Lane:     lane,
				Body:     ticketBodyTemplate(),
			}
			p, err := b.Write(t)
			if err != nil {
				return err
			}
			_ = store.NewLedger(ws.Layout).Append(model.Entry{
				Agent: "human:operator", Ticket: id, Action: model.ActNote,
				Why: "ticket created: " + title,
			})
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s in %s/\n%s\n", id, lane, p)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "ticket title (required)")
	cmd.Flags().StringVar(&role, "role", "", "responsible role id")
	cmd.Flags().StringVar(&priority, "priority", "", "P0|P1|P2|P3")
	cmd.Flags().StringVar(&epic, "epic", "", "parent epic id")
	return cmd
}

func newTicketClaimCmd() *cobra.Command {
	var agent, role, why string
	cmd := &cobra.Command{
		Use:   "claim <id>",
		Short: "Claim a ticket (move to the active lane, set assignee + lock)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if agent == "" {
				return fmt.Errorf("--agent is required")
			}
			b := store.NewBoard(ws.Layout)
			t, ok, err := b.Find(args[0], ws.Workflow.LaneIDs())
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("ticket %s not found", args[0])
			}
			to := claimTarget(ws, t.Lane)
			if to == "" {
				return fmt.Errorf("no claim transition from lane %q", t.Lane)
			}
			if role == "" {
				role = t.Role
			}
			if why == "" {
				why = "claim by " + agent
			}
			res, err := wf.Move(ws, b, store.NewLedger(ws.Layout), args[0], wf.MoveOptions{
				To: to, ByRole: role, Agent: agent, Why: why,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s claimed by %s → %s/\n", res.Ticket.ID, agent, res.To)
			return nil
		},
	}
	cmd.Flags().StringVar(&agent, "agent", "", "agent id claiming the ticket (required)")
	cmd.Flags().StringVar(&role, "role", "", "role to hold (defaults to ticket role)")
	cmd.Flags().StringVar(&why, "why", "", "reason for the ledger")
	return cmd
}

// claimTarget finds the lane a claim transition leads to from the given lane.
func claimTarget(ws *config.Workspace, from string) string {
	for _, tr := range ws.Workflow.Transitions {
		if (tr.From == from || tr.From == "*") && tr.Action == model.ActClaim {
			return tr.To
		}
	}
	return ""
}

func newTicketMoveCmd() *cobra.Command {
	var to, by, agent, why, handoff string
	var approve, skipGates bool
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a ticket to another lane (enforces transitions + gates)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if to == "" {
				return fmt.Errorf("--to is required")
			}
			res, err := wf.Move(ws, store.NewBoard(ws.Layout), store.NewLedger(ws.Layout), args[0], wf.MoveOptions{
				To: to, ByRole: by, Agent: agent, Why: why, Handoff: handoff,
				HumanApproved: approve, SkipGates: skipGates,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, o := range res.Outcomes {
				status := "✓"
				if o.Human {
					status = "⊙"
				} else if !o.Passed {
					status = "✗"
				}
				fmt.Fprintf(out, "  %s gate %s\n", status, o.Name)
			}
			fmt.Fprintf(out, "%s: %s/ → %s/ (%s)\n", res.Ticket.ID, res.From, res.To, res.Action)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "target lane (required)")
	cmd.Flags().StringVar(&by, "by", "", "acting role")
	cmd.Flags().StringVar(&agent, "agent", "", "acting agent")
	cmd.Flags().StringVar(&why, "why", "", "reason for the ledger (required)")
	cmd.Flags().StringVar(&handoff, "handoff", "", "handoff note (required for handoff transitions)")
	cmd.Flags().BoolVar(&approve, "approve", false, "acknowledge human/checklist gates")
	cmd.Flags().BoolVar(&skipGates, "skip-gates", false, "bypass gate evaluation (recorded in ledger)")
	return cmd
}

func newTicketShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Print a ticket",
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
			raw, err := os.ReadFile(b.Path(t))
			if err != nil {
				return err
			}
			cmd.OutOrStdout().Write(raw)
			return nil
		},
	}
	return cmd
}

func ticketBodyTemplate() string {
	return `## Bối cảnh
TODO

## Yêu cầu
- TODO

## Acceptance
- [ ] TODO

## Handoff notes
<!-- mỗi lần đổi assignee/role, thêm 1 mục: đã làm gì, còn gì, cần gì -->
`
}
