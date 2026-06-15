package paths

import (
	"strings"
	"testing"
)

func TestSafeSegmentBlocksTraversal(t *testing.T) {
	cases := map[string]string{
		"T-001":      "T-001",
		"../../etc":  "..-..-etc", // slashes neutralized
		"..":         "_",
		".":          "_",
		"a/b\\c":     "a-b-c",
		"x;rm -rf /": "x-rm--rf--",
	}
	for in, want := range cases {
		if got := SafeSegment(in); got != want {
			t.Errorf("SafeSegment(%q) = %q, want %q", in, got, want)
		}
		if strings.ContainsAny(SafeSegment(in), `/\`) {
			t.Errorf("SafeSegment(%q) still has a separator", in)
		}
	}
}

func TestRunsStaysInsideWorkspace(t *testing.T) {
	l := At("/repo")
	p := l.Runs("../../escape")
	if !strings.HasPrefix(p, "/repo/.hatch/runs/") {
		t.Fatalf("Runs escaped workspace: %s", p)
	}
}
