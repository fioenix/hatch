package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// RunOptions parameterise an orchestrated run.
type RunOptions struct {
	DryRun    bool          // print the invocation, don't execute
	Timeout   time.Duration // 0 = no timeout
	Stdout    io.Writer     // live output sink (defaults to os.Stdout)
	SkipComms bool          // don't prepend inbox + conversation recall
}

// RunOutcome reports the result of a run.
type RunOutcome struct {
	Invocation Invocation
	Executed   bool
	ExitCode   int
	Output     string
	Err        error
}

// Run builds the ticket prompt and executes the agent for that ticket. Like a
// teammate starting work, the agent first "reads the room": its unread inbox
// plus a recall of conversation relevant to the ticket are prepended (unless
// SkipComms), and the inbox is marked read after a successful run.
func Run(ws *config.Workspace, agent model.Agent, t model.Ticket, role string, opt RunOptions) (*RunOutcome, error) {
	prompt := BuildPrompt(t, role)
	if !opt.SkipComms {
		if comm := commContext(ws, agent, t.Title); comm != "" {
			prompt = comm + "\n\n" + prompt
		}
	}
	out, err := Execute(ws, agent, t.ID, prompt, opt)
	if err == nil && out != nil && out.Executed && !opt.SkipComms {
		_ = bus.New(ws.Layout).MarkRead(agent.ID)
	}
	return out, err
}

// commContext gathers the agent's unread inbox and a query-scoped recall of
// recent conversation, formatted as a compact, token-bounded block.
func commContext(ws *config.Workspace, agent model.Agent, query string) string {
	b := bus.New(ws.Layout)
	inbox, _ := b.Inbox(agent.ID, agent.Roles)
	subs := b.Subscriptions(agent.ID)
	recall, _ := b.Search(bus.SearchOpts{Query: query, Channels: subs, Limit: 5})

	if len(inbox) == 0 && len(recall) == 0 {
		return ""
	}
	var s strings.Builder
	s.WriteString("## Hộp thư & bối cảnh trao đổi (đọc nhanh trước khi vào việc)\n")
	if len(inbox) > 0 {
		s.WriteString("\n### Inbox — cần để ý (DM/@mention/broadcast)\n")
		for _, m := range capMsgs(inbox, 10) {
			fmt.Fprintf(&s, "- [%s] %s · %s: %s\n", m.Type, m.Channel, m.From, snippet(m.Body))
		}
	}
	if len(recall) > 0 {
		s.WriteString("\n### Liên quan tới ticket (recall, không cần trả lời hết)\n")
		for _, m := range recall {
			fmt.Fprintf(&s, "- %s · %s: %s\n", m.Channel, m.From, snippet(m.Body))
		}
	}
	s.WriteString("\nTrả lời/ghi nhận nếu @mention đích danh; còn lại chỉ là bối cảnh.")
	return s.String()
}

func capMsgs(ms []bus.Message, n int) []bus.Message {
	if len(ms) > n {
		return ms[len(ms)-n:]
	}
	return ms
}

func snippet(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}

// Execute builds and (unless DryRun) runs an agent against an arbitrary prompt,
// recording the outcome in the ledger under ticketID ("-" for system tasks).
func Execute(ws *config.Workspace, agent model.Agent, ticketID, prompt string, opt RunOptions) (*RunOutcome, error) {
	repoRoot := ws.Layout.RepoRoot()
	req := RunRequest{
		Agent:    agent,
		Prompt:   prompt,
		WorkDir:  repoRoot,
		RepoRoot: repoRoot,
	}
	inv := AdapterFor(agent.Kind).Build(req)
	t := model.Ticket{ID: ticketID}
	out := opt.Stdout
	if out == nil {
		out = os.Stdout
	}
	lg := store.NewLedger(ws.Layout)

	// No headless contract: emit a handoff for a human/IDE and stop.
	if !inv.Headless {
		fmt.Fprintf(out, "Agent %s (%s) has no headless mode: %s\n", agent.ID, agent.Kind, inv.Note)
		fmt.Fprintln(out, "--- handoff prompt ---")
		fmt.Fprintln(out, inv.StdinStr)
		_ = lg.Append(model.Entry{
			Agent: agent.ID, Ticket: t.ID, Action: model.ActNote,
			Why: "manual handoff prepared (" + inv.Note + ")",
		})
		return &RunOutcome{Invocation: inv, Executed: false}, nil
	}

	if opt.DryRun {
		fmt.Fprintln(out, "[dry-run] would run:")
		fmt.Fprintln(out, "  "+strings.Join(redactPrompt(inv.Args), " "))
		if inv.Note != "" {
			fmt.Fprintln(out, "  note: "+inv.Note)
		}
		return &RunOutcome{Invocation: inv, Executed: false}, nil
	}

	// Execute.
	ctx := context.Background()
	if opt.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, inv.Args[0], inv.Args[1:]...)
	cmd.Dir = req.WorkDir
	cmd.Env = append(os.Environ(), inv.Env...)
	if inv.StdinStr != "" {
		cmd.Stdin = strings.NewReader(inv.StdinStr)
	}
	var buf strings.Builder
	cmd.Stdout = io.MultiWriter(out, &buf)
	cmd.Stderr = io.MultiWriter(out, &buf)

	_ = lg.Append(model.Entry{
		Agent: agent.ID, Ticket: t.ID, Action: model.ActStart,
		Why: fmt.Sprintf("orchestrator spawned %s for %s", agent.ID, t.ID),
	})

	runErr := cmd.Run()
	outcome := &RunOutcome{Invocation: inv, Executed: true, Output: buf.String(), Err: runErr}
	if cmd.ProcessState != nil {
		outcome.ExitCode = cmd.ProcessState.ExitCode()
	}

	result := "ok"
	if runErr != nil {
		result = "failed: " + runErr.Error()
	}
	_ = lg.Append(model.Entry{
		Agent: agent.ID, Ticket: t.ID, Action: model.ActProgress,
		Result: result, Why: fmt.Sprintf("%s finished (exit %d)", agent.ID, outcome.ExitCode),
	})
	return outcome, nil
}

// redactPrompt shortens the (often long) prompt argument for display.
func redactPrompt(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if len(a) > 80 && strings.Contains(a, "\n") {
			out[i] = fmt.Sprintf("<prompt %d chars>", len(a))
		} else {
			out[i] = a
		}
	}
	return out
}
