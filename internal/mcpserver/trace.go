package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TraceEntry is one line in the MCP log (.hatch/logs/mcp.jsonl): one tool call,
// who made it, whether it succeeded, and how long it took.
type TraceEntry struct {
	TS    string `json:"ts"`
	Agent string `json:"agent"`
	Tool  string `json:"tool"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	MS    int64  `json:"ms"`
}

var traceMu sync.Mutex

// traceMiddleware logs every tools/call the server handles to path as JSON
// lines. Logging never affects the call: write failures are swallowed.
func traceMiddleware(path, agent string) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			tool := ""
			if r, ok := req.(*mcp.CallToolRequest); ok && r.Params != nil {
				tool = r.Params.Name
			}
			start := time.Now()
			res, err := next(ctx, method, req)

			e := TraceEntry{
				TS:    start.Format(time.RFC3339),
				Agent: agent,
				Tool:  tool,
				OK:    err == nil,
				MS:    time.Since(start).Milliseconds(),
			}
			if err != nil {
				e.Error = err.Error()
			} else if cr, ok := res.(*mcp.CallToolResult); ok && cr != nil && cr.IsError {
				e.OK = false
				e.Error = toolErrorText(cr)
			}
			appendTrace(path, e)
			return res, err
		}
	}
}

func toolErrorText(cr *mcp.CallToolResult) string {
	for _, c := range cr.Content {
		if tc, ok := c.(*mcp.TextContent); ok && tc.Text != "" {
			return tc.Text
		}
	}
	return "tool returned isError"
}

func appendTrace(path string, e TraceEntry) {
	if path == "" {
		return
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	traceMu.Lock()
	defer traceMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(b, '\n'))
}
