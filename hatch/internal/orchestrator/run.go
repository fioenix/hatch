package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// RunOptions parameterise an orchestrated run.
type RunOptions struct {
	DryRun  bool          // print the invocation, don't execute
	Timeout time.Duration // 0 = no timeout
	Stdout  io.Writer     // live output sink (defaults to os.Stdout)
}

// RunOutcome reports the result of a run.
type RunOutcome struct {
	Invocation Invocation
	Executed   bool
	ExitCode   int
	Output     string
	Err        error
}

// Run builds the ticket prompt and executes the agent for that ticket.
func Run(ws *config.Workspace, agent model.Agent, t model.Ticket, role string, opt RunOptions) (*RunOutcome, error) {
	return Execute(ws, agent, t.ID, BuildPrompt(t, role), opt)
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
