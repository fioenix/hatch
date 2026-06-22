package daemon

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

// ExecRunner is the production Runner: it resumes a teammate's CLI session
// headlessly and injects the triggering messages as the prompt. Continuity is
// the member's own session (its memory); the payload is just "here's what was
// said to you while you were away — read the room and act."
type ExecRunner struct {
	RepoRoot string
	Stdout   io.Writer // live output sink (defaults to os.Stdout)
}

// Wake builds and runs the per-kind invocation. A kind with no verified
// headless contract is a no-op here: that teammate is an interactive seat and
// will see the @mention via its own inbox when it next takes a turn.
func (r ExecRunner) Wake(m model.Member, payload []model.Message) error {
	argv, headless := invocation(m, renderPayload(payload))
	if !headless {
		return nil
	}
	out := r.Stdout
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintf(out, "\n— waking %s (%s) with %d message(s) —\n", m.ID, m.Kind, len(payload))
	cmd := exec.Command(argv[0], argv[1:]...)
	if r.RepoRoot != "" {
		cmd.Dir = r.RepoRoot
	}
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// invocation maps a member to its resume-exec argv. Returns headless=false when
// the kind has no confirmed headless+resume contract (agy/kiro/manual): those
// are interactive seats, woken by the human, not by the daemon.
func invocation(m model.Member, prompt string) (argv []string, headless bool) {
	switch m.Kind {
	case "claude":
		// claude -p runs one headless turn; --resume continues the same session
		// so the teammate keeps its memory.
		if m.SessionID != "" {
			return []string{"claude", "-p", "--resume", m.SessionID, prompt}, true
		}
		return []string{"claude", "-p", prompt}, true
	case "codex":
		// codex exec runs one non-interactive turn; `exec resume <id>` continues
		// the same session so the teammate keeps its memory.
		if m.SessionID != "" {
			return []string{"codex", "exec", "resume", m.SessionID, prompt}, true
		}
		return []string{"codex", "exec", prompt}, true
	case "mock":
		return []string{"true"}, true
	default:
		return nil, false // agy, kiro, manual, user: interactive / not driven here
	}
}

// renderPayload turns the triggering messages into a compact prompt telling the
// woken teammate what was said and where, so it can read the thread and act.
func renderPayload(payload []model.Message) string {
	var b strings.Builder
	b.WriteString("Bạn vừa được nhắc tên trong chat của squad. Đọc các tin dưới đây, mở thread liên quan qua MCP (chat_read) để hiểu đầy đủ, rồi hành động hoặc trả lời thẳng trong thread. Giữ đúng Working Agreement: own việc mình bị nhờ, close the loop, không tin rỗng.\n\n")
	for _, m := range payload {
		ch := m.Channel
		b.WriteString(fmt.Sprintf("- [%s] %s → %s: %s\n", ch, m.From, strings.Join(m.To, ","), oneLine(m.Body)))
	}
	return b.String()
}

func oneLine(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len([]rune(s)) > 300 {
		s = string([]rune(s)[:300]) + "…"
	}
	return s
}
