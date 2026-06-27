package daemon

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/fioenix/hatch/internal/model"
	"github.com/fioenix/hatch/internal/session"
)

// ExecRunner is the production Runner: it resumes a teammate's CLI session
// headlessly and injects the triggering messages as the prompt. Continuity is
// the member's own per-thread session (its warm memory); the payload is just
// "here's what was said to you while you were away — read the room and act."
type ExecRunner struct {
	RepoRoot string
	Stdout   io.Writer      // live output sink (defaults to os.Stdout)
	Sessions *session.Store // per-(agent,thread) session store; nil = no warm sessions
}

// wakePlan is how to invoke a teammate for one wake: the argv, whether it is
// headless at all, whether to capture a session id from stdout (codex), and the
// id we assigned up front (claude). Exactly one of capture/assignID is set for
// a fresh warm session; a resume sets neither.
type wakePlan struct {
	argv     []string
	headless bool
	capture  bool
	assignID string
}

// Wake plans the invocation for the payload's thread, runs it, and commits the
// resulting session id so the next wake on that thread resumes warm.
func (r ExecRunner) Wake(m model.Member, payload []model.Message) error {
	thread := threadOf(payload)
	var prior model.Session
	if r.Sessions != nil {
		prior, _ = r.Sessions.Get(m.ID, thread)
	}
	plan := planWake(m, thread, prior, renderPayload(payload))
	if !plan.headless {
		return nil
	}
	out := r.Stdout
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintf(out, "\n— waking %s (%s) on thread %s with %d message(s) —\n", m.ID, m.Kind, thread, len(payload))

	stdout := out
	var cap *sessionCapture
	if plan.capture {
		cap = &sessionCapture{out: out}
		stdout = cap
	}
	cmd := exec.Command(plan.argv[0], plan.argv[1:]...)
	if r.RepoRoot != "" {
		cmd.Dir = r.RepoRoot
	}
	cmd.Stdout = stdout
	cmd.Stderr = out
	runErr := cmd.Run()

	if r.Sessions != nil {
		r.commit(m, thread, prior, plan, cap, runErr)
	}
	return runErr
}

// commit records the session outcome: a fresh assign/capture stores a live
// session, a successful resume bumps its timestamp, and a failed resume marks
// it stale so the next wake starts fresh. Stateless kinds store nothing.
func (r ExecRunner) commit(m model.Member, thread string, prior model.Session, plan wakePlan, cap *sessionCapture, runErr error) {
	switch {
	case plan.assignID != "" && runErr == nil:
		_ = r.Sessions.Put(model.Session{
			Agent: m.ID, Thread: thread, Kind: m.Kind, ID: plan.assignID,
			Status: model.SessionLive, StartedAt: session.Now(), LastResumedAt: session.Now(),
		})
	case plan.capture && runErr == nil && cap != nil && cap.id != "":
		_ = r.Sessions.Put(model.Session{
			Agent: m.ID, Thread: thread, Kind: m.Kind, ID: cap.id,
			Status: model.SessionLive, StartedAt: session.Now(), LastResumedAt: session.Now(),
		})
	case prior.ID != "" && prior.Status == model.SessionLive:
		if runErr != nil {
			_ = r.Sessions.MarkStale(m.ID, thread)
		} else {
			prior.LastResumedAt = session.Now()
			_ = r.Sessions.Put(prior)
		}
	}
}

// planWake maps a member + its prior session to the exec plan. A live prior
// session resumes warm; otherwise a fresh session is started, assigning an id
// (claude) or capturing one (codex). agy/kiro run stateless (no id contract);
// manual/user are interactive seats and not driven here.
func planWake(m model.Member, thread string, prior model.Session, prompt string) wakePlan {
	warm := prior.ID != "" && prior.Status == model.SessionLive
	switch m.Kind {
	case "claude":
		// claude assigns its own session id; --resume continues it warm.
		if warm {
			return wakePlan{argv: []string{"claude", "-p", "--resume", prior.ID, prompt}, headless: true}
		}
		id := uuid4()
		return wakePlan{argv: []string{"claude", "-p", "--session-id", id, prompt}, headless: true, assignID: id}
	case "codex":
		// codex emits its session id in --json (session_meta.id); capture it on
		// the first turn, then `exec resume <id>` continues warm.
		if warm {
			return wakePlan{argv: []string{"codex", "exec", "resume", prior.ID, prompt}, headless: true}
		}
		return wakePlan{argv: []string{"codex", "exec", "--json", prompt}, headless: true, capture: true}
	case "agy":
		// agy exposes no session id → stateless: read the thread + KB each wake.
		return wakePlan{argv: []string{"agy", "-p", prompt}, headless: true}
	case "kiro":
		// kiro-cli chat --no-interactive prints one turn; stateless for now.
		return wakePlan{argv: []string{"kiro-cli", "chat", "--no-interactive", prompt}, headless: true}
	case "mock":
		return wakePlan{argv: []string{"true"}, headless: true}
	default:
		return wakePlan{headless: false} // manual, user: interactive / not driven here
	}
}

// threadOf returns the bus channel the wake is about: the most recent message's
// channel. Empty when there is no payload.
func threadOf(payload []model.Message) string {
	if len(payload) == 0 {
		return ""
	}
	return payload[len(payload)-1].Channel
}

// uuid4 returns a random RFC-4122 v4 UUID (claude requires a valid UUID for
// --session-id). Uses crypto/rand; no external dependency.
func uuid4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// sessionCapture forwards a child's stdout to out while scanning JSONL for the
// codex "session_meta" event to capture its session id. Best-effort: if the
// format drifts and nothing is captured, the wake just isn't stored (the next
// wake starts fresh) — never an error.
type sessionCapture struct {
	out io.Writer
	buf []byte
	id  string
}

func (c *sessionCapture) Write(p []byte) (int, error) {
	if c.id == "" {
		c.buf = append(c.buf, p...)
		for {
			i := bytes.IndexByte(c.buf, '\n')
			if i < 0 {
				break
			}
			c.scan(c.buf[:i])
			c.buf = c.buf[i+1:]
			if c.id != "" {
				c.buf = nil
				break
			}
		}
		if len(c.buf) > 1<<20 { // bound growth if a line never terminates
			c.buf = nil
		}
	}
	return c.out.Write(p)
}

func (c *sessionCapture) scan(line []byte) {
	var e struct {
		Type    string `json:"type"`
		Payload struct {
			ID string `json:"id"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &e) == nil && e.Type == "session_meta" && e.Payload.ID != "" {
		c.id = e.Payload.ID
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
