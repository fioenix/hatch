package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// newGuardCmd is a PreToolUse hook: it denies edits to files matched by
// registry policy.protect (e.g. the SSOT charter/registry), turning the
// governance prose into real enforcement across agents. It is FAIL-OPEN — any
// missing input, unparseable payload, absent workspace, or unrecognised tool
// results in "allow", so it never blocks legitimate work it can't reason about.
func newGuardCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "PreToolUse hook: deny edits to policy-protected files (fail-open)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			allow := func() error {
				if format == "agy" {
					fmt.Fprintln(out, `{"decision":"allow"}`)
				}
				return nil // claude/codex: exit 0, no output = allow
			}

			raw, _ := io.ReadAll(os.Stdin)
			path := editTargetPath(raw)
			if path == "" {
				return allow()
			}
			ws, err := loadWorkspace()
			if err != nil {
				return allow()
			}
			rel := path
			if r, err := filepath.Rel(ws.Layout.RepoRoot(), path); err == nil {
				rel = r
			}
			for _, glob := range ws.Registry.Policy.ProtectGlobs {
				if protectedMatch(glob, rel) {
					reason := fmt.Sprintf("Hatch policy: %q is protected (policy.protect: %q). Sửa qua SSOT + đề xuất trong chat thay vì sửa trực tiếp.", rel, glob)
					return emitDeny(out, format, reason)
				}
			}
			return allow()
		},
	}
	cmd.Flags().StringVar(&format, "format", "claude", "output: claude (Claude/Codex permissionDecision) | agy (decision)")
	return cmd
}

// editTargetPath pulls the target file path from a PreToolUse payload, covering
// the Claude/Codex shape (tool_input.file_path) and the agy shape
// (toolCall.args.TargetFile). Returns "" when there is no file target.
func editTargetPath(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var in struct {
		ToolInput struct {
			FilePath string `json:"file_path"`
		} `json:"tool_input"`
		ToolCall struct {
			Args struct {
				TargetFile string `json:"TargetFile"`
			} `json:"args"`
		} `json:"toolCall"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return ""
	}
	if in.ToolInput.FilePath != "" {
		return in.ToolInput.FilePath
	}
	return in.ToolCall.Args.TargetFile
}

// protectedMatch reports whether a repo-relative path is covered by a protect
// glob. Supports an exact match, filepath.Match patterns, and a trailing "/" or
// "/**" treated as a directory prefix.
func protectedMatch(glob, rel string) bool {
	rel = filepath.ToSlash(rel)
	glob = filepath.ToSlash(glob)
	if rel == glob {
		return true
	}
	if dir := strings.TrimSuffix(strings.TrimSuffix(glob, "**"), "/"); dir != glob {
		return rel == dir || strings.HasPrefix(rel, dir+"/")
	}
	if ok, _ := filepath.Match(glob, rel); ok {
		return true
	}
	return false
}

// emitDeny writes a tool-blocking decision in the agent's hook output format.
func emitDeny(out io.Writer, format, reason string) error {
	if format == "agy" {
		return json.NewEncoder(out).Encode(map[string]any{"decision": "deny", "reason": reason})
	}
	return json.NewEncoder(out).Encode(map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		},
	})
}
