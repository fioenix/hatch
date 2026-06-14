package orchestrator

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// transcriptDir is where per-run logs live, under .hatch/runs/<ticket>/.
func transcriptDir(l paths.Layout, ticket string) string {
	if ticket == "" || ticket == "-" {
		ticket = "system"
	}
	return filepath.Join(l.Root, "runs", ticket)
}

// openTranscript creates an append log file for a run and writes a header.
func openTranscript(l paths.Layout, ticket, agent string) (*os.File, error) {
	dir := transcriptDir(l, ticket)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	ts := time.Now().Format("20060102-150405")
	f, err := os.OpenFile(filepath.Join(dir, ts+"-"+agent+".log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	f.WriteString("# run " + agent + " · " + ticket + " · " + time.Now().Format(time.RFC3339) + "\n")
	return f, nil
}
