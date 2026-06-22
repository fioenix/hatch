package daemon

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

func TestInvocation(t *testing.T) {
	cases := []struct {
		name     string
		member   model.Member
		want     []string
		headless bool
	}{
		{"claude fresh", model.Member{Kind: "claude"}, []string{"claude", "-p", "P"}, true},
		{"claude resume", model.Member{Kind: "claude", SessionID: "s1"}, []string{"claude", "-p", "--resume", "s1", "P"}, true},
		{"codex fresh", model.Member{Kind: "codex"}, []string{"codex", "exec", "P"}, true},
		{"codex resume", model.Member{Kind: "codex", SessionID: "abc"}, []string{"codex", "exec", "resume", "abc", "P"}, true},
		{"mock", model.Member{Kind: "mock"}, []string{"true"}, true},
		{"agy not headless", model.Member{Kind: "agy"}, nil, false},
		{"unknown not headless", model.Member{Kind: "weird"}, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			argv, headless := invocation(c.member, "P")
			if headless != c.headless {
				t.Fatalf("headless: want %v got %v", c.headless, headless)
			}
			if strings.Join(argv, " ") != strings.Join(c.want, " ") {
				t.Fatalf("argv: want %v got %v", c.want, argv)
			}
		})
	}
}
