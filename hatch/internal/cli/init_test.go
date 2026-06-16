package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitignore(t *testing.T) {
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")

	// Created when absent.
	added, err := ensureGitignore(dir, "/.hatch/")
	if err != nil || !added {
		t.Fatalf("first add: added=%v err=%v", added, err)
	}
	if b, _ := os.ReadFile(gi); strings.Count(string(b), "/.hatch/") != 1 {
		t.Fatalf("want one /.hatch/ entry, got: %q", b)
	}

	// Idempotent: second call is a no-op.
	added, err = ensureGitignore(dir, "/.hatch/")
	if err != nil || added {
		t.Fatalf("second add should be no-op: added=%v err=%v", added, err)
	}

	// Appends to an existing file without a trailing newline, preserving content.
	os.WriteFile(gi, []byte("node_modules/"), 0o644)
	if _, err := ensureGitignore(dir, "/.hatch/"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(gi)
	if got := string(b); !strings.Contains(got, "node_modules/") || !strings.HasSuffix(got, "/.hatch/\n") {
		t.Fatalf("append wrong: %q", got)
	}
}
