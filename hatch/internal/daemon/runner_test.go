package daemon

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

func TestPlanWake(t *testing.T) {
	live := func(id string) model.Session { return model.Session{ID: id, Status: model.SessionLive} }

	cases := []struct {
		name     string
		member   model.Member
		prior    model.Session
		want     []string // exact argv, or nil to skip exact match (assign uses a random uuid)
		headless bool
		capture  bool
		assign   bool
	}{
		{"claude resume", model.Member{Kind: "claude"}, live("s1"), []string{"claude", "-p", "--resume", "s1", "P"}, true, false, false},
		{"codex fresh capture", model.Member{Kind: "codex"}, model.Session{}, []string{"codex", "exec", "--json", "P"}, true, true, false},
		{"codex resume", model.Member{Kind: "codex"}, live("abc"), []string{"codex", "exec", "resume", "abc", "P"}, true, false, false},
		{"agy stateless", model.Member{Kind: "agy"}, live("ignored"), []string{"agy", "-p", "P"}, true, false, false},
		{"kiro stateless", model.Member{Kind: "kiro"}, model.Session{}, []string{"kiro-cli", "chat", "--no-interactive", "P"}, true, false, false},
		{"mock", model.Member{Kind: "mock"}, model.Session{}, []string{"true"}, true, false, false},
		{"manual not headless", model.Member{Kind: "manual"}, model.Session{}, nil, false, false, false},
		{"claude fresh assigns", model.Member{Kind: "claude"}, model.Session{}, nil, true, false, true},
		{"claude stale prior → fresh", model.Member{Kind: "claude"}, model.Session{ID: "old", Status: model.SessionStale}, nil, true, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := planWake(c.member, "t-1", c.prior, "P")
			if p.headless != c.headless {
				t.Fatalf("headless: want %v got %v", c.headless, p.headless)
			}
			if p.capture != c.capture {
				t.Errorf("capture: want %v got %v", c.capture, p.capture)
			}
			if c.want != nil && strings.Join(p.argv, " ") != strings.Join(c.want, " ") {
				t.Fatalf("argv: want %v got %v", c.want, p.argv)
			}
			if c.assign {
				if p.assignID == "" || len(p.assignID) != 36 || strings.Count(p.assignID, "-") != 4 {
					t.Fatalf("want assigned uuid, got %q", p.assignID)
				}
				want := []string{"claude", "-p", "--session-id", p.assignID, "P"}
				if strings.Join(p.argv, " ") != strings.Join(want, " ") {
					t.Fatalf("assign argv: want %v got %v", want, p.argv)
				}
			} else if p.assignID != "" {
				t.Errorf("unexpected assignID %q", p.assignID)
			}
		})
	}
}

func TestSessionCapture(t *testing.T) {
	var sink bytes.Buffer
	c := &sessionCapture{out: &sink}
	// codex --json stream: session_meta carries the id; other events do not.
	stream := `{"type":"task_started","payload":{"turn_id":"x"}}
{"type":"session_meta","payload":{"id":"019d718a-bc66-7f60-8ca5-0c623c49ea21","cwd":"/repo"}}
{"type":"event_msg","payload":{"text":"working"}}
`
	// write in two chunks to exercise partial-line buffering
	mid := len(stream) / 2
	_, _ = c.Write([]byte(stream[:mid]))
	_, _ = c.Write([]byte(stream[mid:]))

	if c.id != "019d718a-bc66-7f60-8ca5-0c623c49ea21" {
		t.Fatalf("captured id = %q", c.id)
	}
	if sink.String() != stream {
		t.Errorf("capture must forward all bytes unchanged")
	}
}
