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
	pats := []string{"/.hatch/board/", "/.hatch/ledger/"}

	// Created when absent: adds the header + all patterns.
	n, err := ensureGitignore(dir, "# hdr", pats)
	if err != nil || n != 2 {
		t.Fatalf("first add: n=%d err=%v", n, err)
	}
	b, _ := os.ReadFile(gi)
	if got := string(b); strings.Count(got, "# hdr") != 1 || !strings.Contains(got, "/.hatch/board/") {
		t.Fatalf("missing header/pattern: %q", got)
	}

	// Idempotent: all present → no-op.
	if n, err := ensureGitignore(dir, "# hdr", pats); err != nil || n != 0 {
		t.Fatalf("second add should be no-op: n=%d err=%v", n, err)
	}

	// Adds only the missing pattern, without duplicating the header.
	n, err = ensureGitignore(dir, "# hdr", append(pats, "/.hatch/mcp/"))
	if err != nil || n != 1 {
		t.Fatalf("partial add: n=%d err=%v", n, err)
	}
	if b, _ := os.ReadFile(gi); strings.Count(string(b), "# hdr") != 1 {
		t.Fatalf("header duplicated: %q", b)
	}

	// Appends to an existing file with no trailing newline, preserving content.
	os.WriteFile(gi, []byte("node_modules/"), 0o644)
	if _, err := ensureGitignore(dir, "# hdr", []string{"/.hatch/board/"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(gi); !strings.Contains(string(b), "node_modules/") || !strings.HasSuffix(string(b), "/.hatch/board/\n") {
		t.Fatalf("append wrong: %q", b)
	}
}
