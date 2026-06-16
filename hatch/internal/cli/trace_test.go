package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTrace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "mcp.jsonl")
	os.WriteFile(p, []byte(
		`{"ts":"2026-06-16T17:40:32+07:00","agent":"codex","tool":"whoami","ok":true,"ms":1}`+"\n"+
			`{"ts":"2026-06-16T17:40:33+07:00","agent":"codex","tool":"chat_read","ok":false,"error":"missing channel","ms":0}`+"\n"+
			`garbage line`+"\n"+
			`{"ts":"2026-06-16T17:40:34+07:00","agent":"claude-code","tool":"chat_inbox","ok":true,"ms":2}`+"\n",
	), 0o644)

	all := readTrace(p, false)
	if len(all) != 3 { // garbage skipped
		t.Fatalf("want 3 entries, got %d", len(all))
	}
	errs := readTrace(p, true)
	if len(errs) != 1 || errs[0].Tool != "chat_read" {
		t.Fatalf("errors-only wrong: %+v", errs)
	}
	if n := countLines(p); n != 4 {
		t.Errorf("countLines = %d, want 4", n)
	}
	// Missing file → empty, no panic.
	if got := readTrace(filepath.Join(dir, "none.jsonl"), false); got != nil {
		t.Errorf("missing file should be nil, got %v", got)
	}

	line := fmtTrace(errs[0])
	if !strings.Contains(line, "✗") || !strings.Contains(line, "chat_read") || !strings.Contains(line, "missing channel") {
		t.Errorf("fmtTrace wrong: %q", line)
	}
}
