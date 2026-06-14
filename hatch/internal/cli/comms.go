package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
)

func newMsgCmd() *cobra.Command {
	var from, to, channel, thread, replyTo, typ string
	cmd := &cobra.Command{
		Use:   "msg <body>",
		Short: "Post to a channel (Slack-style): DM, @mention, or reply in a thread",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			conv := firstNonEmpty(channel, thread)
			if from == "" || conv == "" {
				return fmt.Errorf("--from and --channel are required")
			}
			// Default audience to the channel itself (a plain channel post).
			recipients := splitCSV(to)
			if len(recipients) == 0 {
				recipients = []string{conv}
			}
			m, err := bus.New(ws.Layout).Post(bus.Message{
				Channel: conv, From: from, To: recipients, Type: typ,
				InReplyTo: replyTo, Body: strings.Join(args, " "),
			})
			if err != nil {
				return err
			}
			where := conv
			if replyTo != "" {
				where += " (reply to " + replyTo + ")"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "posted %s to %s\n", m.ID, where)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "sender (agent id or human:<name>)")
	cmd.Flags().StringVar(&to, "to", "", "recipients/mentions: agent/role/#channel/*, comma-separated")
	cmd.Flags().StringVarP(&channel, "channel", "c", "", "channel/DM/conversation id (e.g. #design, dm-codex-claude, T-123)")
	cmd.Flags().StringVar(&thread, "thread", "", "alias for --channel")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "reply within a thread rooted at this message id")
	cmd.Flags().StringVar(&typ, "type", "msg", "msg | ask | reply | decision")
	return cmd
}

func newChannelCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "channel", Aliases: []string{"chan"}, Short: "List and read channels"}

	ls := &cobra.Command{
		Use:   "ls",
		Short: "List all channels / DMs / conversations",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			chans, err := bus.New(ws.Layout).Channels()
			if err != nil {
				return err
			}
			if len(chans) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no channels yet")
			}
			for _, c := range chans {
				fmt.Fprintln(cmd.OutOrStdout(), c)
			}
			return nil
		},
	}

	var in string
	show := &cobra.Command{
		Use:   "show <channel>",
		Short: "Print a channel (or a single thread with --in <rootId>)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			bs := bus.New(ws.Layout)
			out := cmd.OutOrStdout()
			if in != "" {
				msgs, err := bs.Replies(args[0], in)
				if err != nil {
					return err
				}
				if len(msgs) == 0 {
					fmt.Fprintf(out, "no thread rooted at %s in %s\n", in, args[0])
				}
				for _, m := range msgs {
					fmt.Fprintf(out, "%s · %s → %s · %s\n  %s\n", m.ID, m.From, strings.Join(m.To, ","), m.Type, oneLine(m.Body))
				}
				return nil
			}
			raw, err := bs.Raw(args[0])
			if err != nil {
				return err
			}
			if raw == "" {
				fmt.Fprintf(out, "channel %s is empty\n", args[0])
				return nil
			}
			fmt.Fprint(out, raw)
			return nil
		},
	}
	show.Flags().StringVar(&in, "in", "", "show only the thread rooted at this message id")

	var joinAgent string
	join := &cobra.Command{
		Use:   "join <channel>",
		Short: "Subscribe an agent to a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if joinAgent == "" {
				return fmt.Errorf("--agent is required")
			}
			if err := bus.New(ws.Layout).Subscribe(args[0], joinAgent); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s subscribed to %s\n", joinAgent, args[0])
			return nil
		},
	}
	join.Flags().StringVar(&joinAgent, "agent", "", "agent id to subscribe (required)")

	var leaveAgent string
	leave := &cobra.Command{
		Use:   "leave <channel>",
		Short: "Unsubscribe an agent from a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			if leaveAgent == "" {
				return fmt.Errorf("--agent is required")
			}
			if err := bus.New(ws.Layout).Unsubscribe(args[0], leaveAgent); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s left %s\n", leaveAgent, args[0])
			return nil
		},
	}
	leave.Flags().StringVar(&leaveAgent, "agent", "", "agent id to unsubscribe (required)")

	members := &cobra.Command{
		Use:   "members <channel>",
		Short: "List a channel's subscribers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			ms := bus.New(ws.Layout).Members(args[0])
			if len(ms) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no subscribers")
			}
			for _, a := range ms {
				fmt.Fprintln(cmd.OutOrStdout(), a)
			}
			return nil
		},
	}

	cmd.AddCommand(ls, show, join, leave, members)
	return cmd
}

func newSearchCmd() *cobra.Command {
	var channel, from, typ, agent string
	var limit int
	var all bool
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Recall relevant messages into context (not a firehose; newest-first, capped)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			bs := bus.New(ws.Layout)
			opts := bus.SearchOpts{
				Query: strings.Join(args, " "), Channel: channel, From: from, Type: typ, Limit: limit,
			}
			// Default scope: the agent's subscribed channels (unless --all / --channel).
			if agent != "" && channel == "" && !all {
				subs := bs.Subscriptions(agent)
				if len(subs) > 0 {
					opts.Channels = subs
				}
			}
			hits, err := bs.Search(opts)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(hits) == 0 {
				fmt.Fprintln(out, "no matching messages")
			}
			for _, m := range hits {
				fmt.Fprintf(out, "%s · %s · %s → %s · %s\n  %s\n", m.TS, m.Channel, m.From, strings.Join(m.To, ","), m.Type, oneLine(m.Body))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&channel, "channel", "c", "", "restrict to one channel")
	cmd.Flags().StringVar(&from, "from", "", "restrict to a sender")
	cmd.Flags().StringVar(&typ, "type", "", "restrict to a message type")
	cmd.Flags().StringVar(&agent, "agent", "", "scope to this agent's subscriptions by default")
	cmd.Flags().IntVar(&limit, "limit", 20, "max results (newest first)")
	cmd.Flags().BoolVar(&all, "all", false, "search all channels, ignoring subscriptions")
	return cmd
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func newInboxCmd() *cobra.Command {
	var mark bool
	cmd := &cobra.Command{
		Use:   "inbox <agent>",
		Short: "Show messages addressed to an agent (since its last read)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			agent := args[0]
			var roles []string
			if a, ok := ws.Registry.AgentByID(agent); ok {
				roles = a.Roles
			}
			msgs, err := bus.New(ws.Layout).Inbox(agent, roles)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(msgs) == 0 {
				fmt.Fprintln(out, "inbox empty")
			}
			for _, m := range msgs {
				fmt.Fprintf(out, "[%s] %s · %s → %s\n  %s\n", m.Type, m.Channel, m.From, strings.Join(m.To, ","), oneLine(m.Body))
			}
			if mark {
				if err := bus.New(ws.Layout).MarkRead(agent); err != nil {
					return err
				}
				fmt.Fprintln(out, "(marked read)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&mark, "mark", false, "mark inbox read (advance cursor)")
	return cmd
}

func newThreadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thread <id>",
		Short: "Print a conversation thread (or list threads with no arg)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			bs := bus.New(ws.Layout)
			out := cmd.OutOrStdout()
			if len(args) == 0 {
				threads, err := bs.Channels()
				if err != nil {
					return err
				}
				if len(threads) == 0 {
					fmt.Fprintln(out, "no threads")
				}
				for _, th := range threads {
					fmt.Fprintln(out, th)
				}
				return nil
			}
			raw, err := bs.Raw(args[0])
			if err != nil {
				return err
			}
			if raw == "" {
				fmt.Fprintf(out, "thread %s is empty\n", args[0])
				return nil
			}
			fmt.Fprint(out, raw)
			return nil
		},
	}
	return cmd
}

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
			outcome, err := orchestrator.Execute(ws, target, thread, prompt, orchestrator.RunOptions{
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
	var thread, topic, agentsCSV, chair string
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
			for r := 1; r <= rounds; r++ {
				for _, a := range participants {
					raw, _ := bs.Raw(thread)
					prompt := orchestrator.BuildMeetingPrompt(roleOf(a), thread, topic, raw, r, rounds)
					fmt.Fprintf(out, "\n# round %d · %s (%s)\n", r, a.ID, roleOf(a))
					outcome, err := orchestrator.Execute(ws, a, thread, prompt, orchestrator.RunOptions{
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
					if _, err := bs.Post(bus.Message{Channel: thread, From: a.ID, To: []string{"*"}, Type: turnType(turn), Body: turn}); err != nil {
						return err
					}
				}
			}
			fmt.Fprintf(out, "\nmeeting recorded in thread %s\n", thread)
			return nil
		},
	}
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

func oneLine(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > 100 {
		return s[:100] + "…"
	}
	return s
}
