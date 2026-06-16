package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fioenix/overclaud/hatch/internal/bus"
)

// newBriefCmd prints an agent's "read the room" briefing — its inbox (DMs,
// @mentions, broadcasts) plus the open task threads. It is what an agent's
// session-start lifecycle hook calls so the agent walks in already knowing what
// the squad needs from it. Output is JSON `hookSpecificOutput.additionalContext`
// (the shape Claude Code / Codex / agy hooks inject) or plain --text.
//
// It never fails a session: with no workspace or empty inbox it emits nothing
// and exits 0.
func newBriefCmd() *cobra.Command {
	var as, format string
	cmd := &cobra.Command{
		Use:   "brief",
		Short: "Briefing for an agent (inbox + open threads) — for session-start hooks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			ws, err := loadWorkspace()
			if err != nil {
				return nil // no workspace here → say nothing, don't break the session
			}
			id := as
			if id == "" {
				id = resolveIdentity(ws)
			}
			if id == "" {
				return nil
			}
			var roles []string
			if a, ok := ws.Registry.AgentByID(id); ok {
				roles = a.Roles
			}
			b := bus.New(ws.Layout)
			msgs, _ := b.Inbox(id, roles)
			channels, _ := b.Channels()

			text := briefText(id, roles, msgs, channels)
			if text == "" {
				return nil
			}
			if format == "text" {
				fmt.Fprintln(out, text)
				return nil
			}
			payload := map[string]any{
				"hookSpecificOutput": map[string]any{
					"hookEventName":     "SessionStart",
					"additionalContext": text,
				},
			}
			enc := json.NewEncoder(out)
			return enc.Encode(payload)
		},
	}
	cmd.Flags().StringVar(&as, "as", "", "agent id to brief (default: $HATCH_AGENT or first claude-kind agent)")
	cmd.Flags().StringVar(&format, "format", "json", "output: json (hook additionalContext) | text")
	return cmd
}

// briefText renders the squad briefing, or "" when there is nothing to say.
func briefText(id string, roles []string, msgs []bus.Message, channels []string) string {
	if len(msgs) == 0 && len(channels) == 0 {
		return ""
	}
	var s strings.Builder
	fmt.Fprintf(&s, "Hatch squad — bạn là %s", id)
	if len(roles) > 0 {
		fmt.Fprintf(&s, " (%s)", strings.Join(roles, ", "))
	}
	s.WriteString(". Chat dùng chung là backlog: mỗi thread = một task.\n")

	if len(msgs) > 0 {
		fmt.Fprintf(&s, "\nĐang chờ bạn (%d):\n", len(msgs))
		for _, m := range msgs {
			fmt.Fprintf(&s, "- #%s · %s: %s\n", strings.TrimPrefix(m.Channel, "#"), m.From, oneLine(m.Body))
		}
	}
	if len(channels) > 0 {
		fmt.Fprintf(&s, "\nThread đang mở (%d): %s\n", len(channels), strings.Join(channels, ", "))
	}
	s.WriteString("\nDùng MCP tool `chat_inbox`/`chat_read` để xử lý, `chat_post` để trả lời trong thread, `chat_open` cho task mới.")
	return s.String()
}
