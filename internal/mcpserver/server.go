// Package mcpserver exposes Hatch's shared chat (the bus) and knowledge base to
// any MCP-capable coding agent (Claude Code, Codex, agy, Kiro, …). This is the
// "embedded harness": the agent drives itself and reaches into the shared
// comms + memory through these tools — chat is both the communication channel
// and the backlog (a thread = a task). See docs/20-embedded-harness-pivot.md.
package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fioenix/hatch/internal/bus"
	"github.com/fioenix/hatch/internal/config"
	"github.com/fioenix/hatch/internal/model"
	"github.com/fioenix/hatch/internal/roster"
	"github.com/fioenix/hatch/internal/store"
)

// New builds the Hatch MCP server bound to a workspace, acting as agent `me`.
// All posts/inbox are attributed to `me`, so each agent runs its own instance
// (`hatch mcp --as <agent>`).
func New(ws *config.Workspace, me, version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "hatch", Title: "Hatch chat", Version: version}, nil)
	// Self-observability: record every tool call (who, what, ok/err, latency) to
	// the workspace MCP log, so `hatch trace` shows what agents did and surfaces
	// Hatch's own errors — without digging into each agent's MCP logs.
	s.AddReceivingMiddleware(traceMiddleware(ws.Layout.MCPLog(), me))
	b := bus.New(ws.Layout)
	kb := store.NewKB(ws.Layout)
	rs := roster.New(ws.Layout)
	roles := rolesOf(ws, me)
	kindOf := func(id string) string {
		if a, ok := ws.Registry.AgentByID(id); ok {
			return a.Kind
		}
		return ""
	}
	// touch refreshes presence on any activity, so the roster reflects who is
	// actually around (best-effort; presence errors never fail a tool call).
	touch := func() { _ = rs.Touch(me) }

	mcp.AddTool(s, &mcp.Tool{Name: "whoami",
		Description: "Bạn là agent nào trong squad + giữ vai gì. Gọi đầu session."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, whoamiOut, error) {
			touch()
			return nil, whoamiOut{Agent: me, Roles: roles}, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "join",
		Description: "Vào phòng chung của workspace: đăng ký bạn vào roster (online) để đồng đội biết bạn có mặt. Truyền session_id để teammate đánh thức đúng phiên có trí nhớ của bạn. Gọi đầu session, sau whoami."},
		func(_ context.Context, _ *mcp.CallToolRequest, in joinIn) (*mcp.CallToolResult, joinOut, error) {
			kind := in.Kind
			if kind == "" {
				kind = kindOf(me)
			}
			rl := splitCSV(in.Roles)
			if len(rl) == 0 {
				rl = roles
			}
			m, err := rs.Join(model.Member{ID: me, Kind: kind, Roles: rl, SessionID: in.SessionID, Note: in.Note})
			if err != nil {
				return nil, joinOut{}, err
			}
			return nil, joinOut{ID: m.ID, Status: m.Status}, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "roster",
		Description: "Ai đang ở trong phòng: liệt kê thành viên + vai trò + trạng thái (online/idle/suspended/offline) + last-seen. Xem trước khi nhờ việc để biết gọi ai."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, rosterOut, error) {
			touch()
			r, err := rs.Effective(time.Now())
			if err != nil {
				return nil, rosterOut{}, err
			}
			out := rosterOut{}
			for _, m := range roster.Members(r) {
				rolesStr := strings.Join(m.Roles, ",")
				out.Members = append(out.Members, fmt.Sprintf("%s [%s] %s · %s · seen %s", m.ID, m.Kind, rolesStr, m.Status, m.LastSeen))
			}
			return nil, out, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "leave",
		Description: "Rời phòng: đánh dấu bạn offline (sẽ không bị đánh thức nữa cho tới khi join lại)."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, joinOut, error) {
			if err := rs.Leave(me); err != nil {
				return nil, joinOut{}, err
			}
			return nil, joinOut{ID: me, Status: model.MemberOffline}, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "chat_open",
		Description: "Mở một thread/topic cho MỘT task: post message gốc vào channel (tạo channel nếu chưa có). Dùng @tag trong body để gọi đồng đội. Trả về channel + id message gốc (= id task)."},
		func(_ context.Context, _ *mcp.CallToolRequest, in openIn) (*mcp.CallToolResult, postOut, error) {
			touch()
			ch := in.Channel
			if ch == "" {
				ch = "#" + slug(in.Title)
			}
			body := in.Body
			if in.Title != "" {
				body = "**" + in.Title + "**\n" + body
			}
			m, err := b.Post(bus.Message{Channel: ch, From: me, To: splitCSV(in.To), Type: model.MsgText, Body: body})
			if err != nil {
				return nil, postOut{}, err
			}
			return nil, postOut{Channel: ch, MessageID: m.ID}, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "chat_post",
		Description: "Brief tiến độ/kết quả hoặc trả lời trong một channel. reply_to = id message gốc để nối vào thread đó. @tag trong body để gọi đồng đội."},
		func(_ context.Context, _ *mcp.CallToolRequest, in postIn) (*mcp.CallToolResult, postOut, error) {
			touch()
			if in.Channel == "" {
				return nil, postOut{}, fmt.Errorf("channel is required")
			}
			typ := in.Type
			if typ == "" {
				typ = model.MsgText
			}
			m, err := b.Post(bus.Message{Channel: in.Channel, From: me, To: splitCSV(in.To), Type: typ, InReplyTo: in.ReplyTo, Body: in.Body})
			if err != nil {
				return nil, postOut{}, err
			}
			return nil, postOut{Channel: in.Channel, MessageID: m.ID}, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "chat_read",
		Description: "Đọc toàn bộ hội thoại của một channel/thread (để hiểu nhiệm vụ trước khi phản hồi)."},
		func(_ context.Context, _ *mcp.CallToolRequest, in channelIn) (*mcp.CallToolResult, textOut, error) {
			raw, err := b.Raw(in.Channel)
			return nil, textOut{Text: raw}, err
		})

	mcp.AddTool(s, &mcp.Tool{Name: "chat_inbox",
		Description: "Tin nhắn gửi tới bạn (DM/@mention/broadcast) kể từ lần đọc trước. Gọi để 'đọc phòng' trước khi vào việc. mark=true để đánh dấu đã đọc."},
		func(_ context.Context, _ *mcp.CallToolRequest, in inboxIn) (*mcp.CallToolResult, messagesOut, error) {
			touch()
			msgs, err := b.Inbox(me, roles)
			if err != nil {
				return nil, messagesOut{}, err
			}
			if in.Mark {
				_ = b.MarkRead(me)
			}
			return nil, messagesOut{Messages: brief(msgs)}, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "chat_search",
		Description: "Tra cứu hội thoại liên quan (recall theo từ khoá), không phải đọc hết. newest-first, có giới hạn."},
		func(_ context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, messagesOut, error) {
			lim := in.Limit
			if lim == 0 {
				lim = 20
			}
			msgs, err := b.Search(model.SearchOpts{Query: in.Query, Channel: in.Channel, From: in.From, Type: in.Type, Limit: lim})
			return nil, messagesOut{Messages: brief(msgs)}, err
		})

	mcp.AddTool(s, &mcp.Tool{Name: "chat_channels",
		Description: "Liệt kê các channel/topic/task hiện có (backlog dạng hội thoại)."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, channelsOut, error) {
			chs, err := b.Channels()
			// Present channels with a leading '#' so the surface matches the
			// ids returned by chat_open (the bus stores them bare).
			for i, c := range chs {
				chs[i] = "#" + c
			}
			return nil, channelsOut{Channels: chs}, err
		})

	mcp.AddTool(s, &mcp.Tool{Name: "kb_add",
		Description: "Ghi tri thức đáng giữ vào bộ nhớ chung: type=decision|domain|learning."},
		func(_ context.Context, _ *mcp.CallToolRequest, in kbAddIn) (*mcp.CallToolResult, kbAddOut, error) {
			typ := in.Type
			if typ == "" {
				typ = model.KBLearning
			}
			e := model.KBEntry{ID: kb.NextID(typ), Type: typ, Title: in.Title, Tags: splitCSV(in.Tags), Author: me, Body: in.Body}
			if _, err := kb.Add(e); err != nil {
				return nil, kbAddOut{}, err
			}
			_ = kb.RebuildIndex()
			return nil, kbAddOut{ID: e.ID}, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "kb_search",
		Description: "Tra bộ nhớ chung theo tag trước khi suy diễn lại."},
		func(_ context.Context, _ *mcp.CallToolRequest, in kbSearchIn) (*mcp.CallToolResult, kbSearchOut, error) {
			es, err := kb.Query(splitCSV(in.Tags))
			out := kbSearchOut{}
			for _, e := range es {
				out.Entries = append(out.Entries, fmt.Sprintf("%s [%s] %s — kb/%s", e.ID, e.Type, e.Title, e.Path))
			}
			return nil, out, err
		})

	return s
}

func rolesOf(ws *config.Workspace, id string) []string {
	if a, ok := ws.Registry.AgentByID(id); ok {
		return a.Roles
	}
	return nil
}

func brief(ms []bus.Message) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		body := strings.ReplaceAll(strings.TrimSpace(m.Body), "\n", " ")
		if len([]rune(body)) > 200 {
			body = string([]rune(body)[:200]) + "…"
		}
		out = append(out, fmt.Sprintf("[%s] %s · %s: %s", m.Type, m.Channel, m.From, body))
	}
	return out
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
