package paths

import (
	"os"
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

func TestFindLocalOverridesGlobal(t *testing.T) {
	root := t.TempDir()
	global := root + "/home/.hatch"
	repo := root + "/repo"
	local := repo + "/.hatch"
	for _, d := range []string{global, local, repo + "/sub"} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HATCH_HOME", global)

	// From inside the repo (even a subdir), the local .hatch wins.
	got, err := Find(repo + "/sub")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.Root != local {
		t.Errorf("Find resolved %q, want local %q", got.Root, local)
	}

	// Outside any local workspace, it falls back to the global one.
	got, err = Find(root + "/elsewhere")
	if err != nil {
		t.Fatalf("Find global: %v", err)
	}
	if got.Root != global {
		t.Errorf("Find resolved %q, want global %q", got.Root, global)
	}
}

func TestFindNoWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HATCH_HOME", root+"/nonexistent/.hatch")
	if _, err := Find(root); err != ErrNotFound {
		t.Errorf("Find = %v, want ErrNotFound", err)
	}
}
