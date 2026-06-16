package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// traceLine mirrors mcpserver.TraceEntry — the .hatch/logs/mcp.jsonl format.
type traceLine struct {
	TS    string `json:"ts"`
	Agent string `json:"agent"`
	Tool  string `json:"tool"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	MS    int64  `json:"ms"`
}

// newTraceCmd shows the MCP tool-call log — what each agent did through Hatch,
// and which calls errored. This is Hatch's self-observability: it surfaces
// Hatch's own failures (bad routing, tool errors) in one place instead of
// scattered across each agent's MCP logs.
func newTraceCmd() *cobra.Command {
	var errorsOnly, follow bool
	var n int
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "Show the MCP tool-call log (who called what, ok/err, latency)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := loadWorkspace()
			if err != nil {
				return err
			}
			path := ws.Layout.MCPLog()
			out := cmd.OutOrStdout()

			lines := readTrace(path, errorsOnly)
			if len(lines) == 0 && !follow {
				fmt.Fprintf(out, "Chưa có tool-call nào (%s trống). Chạy agent qua MCP rồi xem lại.\n", path)
				return nil
			}
			if len(lines) > n {
				lines = lines[len(lines)-n:]
			}
			for _, l := range lines {
				fmt.Fprintln(out, fmtTrace(l))
			}
			if !follow {
				return nil
			}
			// Tail: poll for appended lines until interrupted.
			seen := countLines(path)
			for {
				time.Sleep(500 * time.Millisecond)
				if total := countLines(path); total > seen {
					fresh := readTrace(path, errorsOnly)
					for _, l := range fresh[min(seen, len(fresh)):] {
						fmt.Fprintln(out, fmtTrace(l))
					}
					seen = total
				}
			}
		},
	}
	cmd.Flags().BoolVar(&errorsOnly, "errors", false, "show only failed calls")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "tail the log live")
	cmd.Flags().IntVarP(&n, "n", "n", 20, "show the last N entries")
	return cmd
}

func readTrace(path string, errorsOnly bool) []traceLine {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []traceLine
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var l traceLine
		if json.Unmarshal(sc.Bytes(), &l) != nil {
			continue
		}
		if errorsOnly && l.OK {
			continue
		}
		out = append(out, l)
	}
	return out
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		n++
	}
	return n
}

func fmtTrace(l traceLine) string {
	ts := l.TS
	if t, err := time.Parse(time.RFC3339, l.TS); err == nil {
		ts = t.Format("15:04:05")
	}
	mark := "✓"
	if !l.OK {
		mark = "✗"
	}
	s := fmt.Sprintf("%s  %-12s %-14s %s %dms", ts, l.Agent, l.Tool, mark, l.MS)
	if l.Error != "" {
		s += "  — " + strings.SplitN(l.Error, "\n", 2)[0]
	}
	return s
}
