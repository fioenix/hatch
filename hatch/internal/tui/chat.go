package tui

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
)

// senderPalette colours messages by author so a thread is easy to scan.
var senderPalette = []string{"#7c3aed", "#0ea5e9", "#16a34a", "#d97706", "#db2777", "#0891b2"}

func senderStyle(name string) lipgloss.Style {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	c := senderPalette[int(h.Sum32())%len(senderPalette)]
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c))
}

type chat struct {
	ws       *config.Workspace
	bus      *bus.Bus
	channels []string
	counts   map[string]int
	sel      int
	msgs     viewport.Model
	md       *glamour.TermRenderer // markdown renderer for message bodies
	follow   bool                  // tail: stick to newest message
	showHelp bool
	w, h     int
	ready    bool
}

// NewChat returns a read-only, live Slack-style TUI for watching the squad's
// shared chat. Agents post through the Hatch MCP server; this only observes.
func NewChat(ws *config.Workspace) *tea.Program {
	c := &chat{ws: ws, bus: bus.New(ws.Layout), follow: true, counts: map[string]int{}}
	return tea.NewProgram(c, tea.WithAltScreen())
}

func (c *chat) Init() tea.Cmd { return tick() }

func (c *chat) reload() {
	chs, _ := c.bus.Channels()
	sort.Strings(chs)
	c.channels = chs
	for _, ch := range chs {
		if m, err := c.bus.Messages(ch); err == nil {
			c.counts[ch] = len(m)
		}
	}
	if c.sel >= len(chs) {
		c.sel = max(0, len(chs)-1)
	}
	off := c.msgs.YOffset
	c.msgs.SetContent(c.renderMessages())
	if c.follow {
		c.msgs.GotoBottom()
	} else {
		c.msgs.SetYOffset(off) // preserve scroll position when paused
	}
}

func (c *chat) current() string {
	if c.sel >= 0 && c.sel < len(c.channels) {
		return c.channels[c.sel]
	}
	return ""
}

func (c *chat) renderMessages() string {
	ch := c.current()
	if ch == "" {
		return dim.Render("(chưa có thread — agent mở task qua Hatch MCP `chat_open`)")
	}
	msgs, err := c.bus.Messages(ch)
	if err != nil || len(msgs) == 0 {
		return dim.Render("(thread trống)")
	}
	rule := dim.Render(strings.Repeat("─", max(8, c.msgWidth()-4)))
	var b strings.Builder
	for i, m := range msgs {
		ts := m.TS
		if t, e := time.Parse(time.RFC3339Nano, m.TS); e == nil {
			ts = t.Format("15:04")
		}
		head := senderStyle(m.From).Render(m.From) + "  " + dim.Render(ts)
		if m.Type != bus.TypeMsg {
			head += " " + laneSty.Render(m.Type)
		}
		if len(m.To) > 0 {
			head += dim.Render("  → " + strings.Join(m.To, ", "))
		}
		b.WriteString(head + "\n")
		b.WriteString(c.renderBody(m.Body))
		if i < len(msgs)-1 {
			b.WriteString("\n" + rule + "\n")
		}
	}
	return b.String()
}

func (c *chat) msgWidth() int { return max(20, c.w-c.chanWidth()-4) }

func (c *chat) chanWidth() int {
	w := c.w / 4
	if w < 16 {
		w = 16
	}
	if w > 32 {
		w = 32
	}
	return w
}

func (c *chat) layout() {
	c.msgs = viewport.New(c.msgWidth(), max(3, c.h-5))
	// Markdown renderer sized to the message pane — renders code blocks,
	// headings, bold, lists like a real chat client.
	if r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(20, c.msgWidth()-4)),
	); err == nil {
		c.md = r
	}
	c.ready = true
}

// renderBody renders a message body as markdown, falling back to plain text.
func (c *chat) renderBody(body string) string {
	body = strings.TrimRight(body, "\n")
	if c.md == nil {
		return body
	}
	out, err := c.md.Render(body)
	if err != nil {
		return body
	}
	return strings.Trim(out, "\n")
}

// mcpTraceCounts returns total tool-calls and failed calls from the MCP log.
func mcpTraceCounts(path string) (calls, errs int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		calls++
		if strings.Contains(string(line), `"ok":false`) {
			errs++
		}
	}
	return calls, errs
}

func (c *chat) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.w, c.h = msg.Width, msg.Height
		c.layout()
		c.reload()
	case tickMsg:
		if c.ready {
			c.reload()
		}
		return c, tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return c, tea.Quit
		case "?":
			c.showHelp = !c.showHelp
		case "]", "tab", "right", "l": // next thread
			if c.sel < len(c.channels)-1 {
				c.sel++
				c.follow = true
				c.reload()
			}
		case "[", "shift+tab", "left", "h": // previous thread
			if c.sel > 0 {
				c.sel--
				c.follow = true
				c.reload()
			}
		case "up", "k":
			c.msgs.LineUp(1)
			c.follow = c.msgs.AtBottom()
		case "down", "j":
			c.msgs.LineDown(1)
			c.follow = c.msgs.AtBottom()
		case "pgup", "ctrl+u", "b":
			c.msgs.HalfViewUp()
			c.follow = false
		case "pgdown", "ctrl+d", " ":
			c.msgs.HalfViewDown()
			c.follow = c.msgs.AtBottom()
		case "g", "home":
			c.msgs.GotoTop()
			c.follow = false
		case "G", "end":
			c.msgs.GotoBottom()
			c.follow = true
		case "f": // toggle live tail
			c.follow = !c.follow
			if c.follow {
				c.msgs.GotoBottom()
			}
		}
	}
	return c, nil
}

func (c *chat) View() string {
	if !c.ready {
		return "loading…"
	}
	if c.showHelp {
		return c.helpView()
	}
	project := c.ws.Registry.Project
	if project == "" {
		project = "Hatch"
	}
	live := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#16a34a")).Render("● LIVE")
	if !c.follow {
		live = dim.Render("⏸ PAUSED")
	}
	header := hdr.Render(project+" — chat") + "  " + live + "  " + dim.Render("(read-only · ? help)")

	var cl strings.Builder
	nameW := max(6, c.chanWidth()-7) // room for "▸ " + " (N)" inside the box
	for i, ch := range c.channels {
		cnt := dim.Render(fmt.Sprintf(" (%d)", c.counts[ch]))
		name := trunc(ch, nameW)
		if i == c.sel {
			cl.WriteString(selSty.Render("▸ "+name) + cnt + "\n")
		} else {
			cl.WriteString("  " + name + cnt + "\n")
		}
	}
	if len(c.channels) == 0 {
		cl.WriteString(dim.Render("  (chưa có thread)"))
	}
	chanBox := paneBox(false, "THREADS", cl.String(), c.chanWidth(), c.h-5)
	title := "—"
	if c.current() != "" {
		title = "#" + c.current()
	}
	msgBox := paneBox(true, title, c.msgs.View(), c.msgWidth(), c.h-5)
	body := lipgloss.JoinHorizontal(lipgloss.Top, chanBox, msgBox)

	stats := laneSty.Render("◆ ") + dim.Render(c.statsLine())
	keys := dim.Render("[ ] thread · ↑↓ scroll · G newest · f follow · ? help · q quit")
	return header + "\n" + body + "\n" + stats + "\n" + keys
}

// statsLine summarises the squad (the numbers the old board showed): threads,
// total messages, roster size, and MCP tool-call activity from the trace log.
func (c *chat) statsLine() string {
	msgs := 0
	for _, ch := range c.channels {
		msgs += c.counts[ch]
	}
	calls, errs := mcpTraceCounts(c.ws.Layout.MCPLog())
	s := fmt.Sprintf("%d threads · %d msgs · %d agents · %d MCP calls",
		len(c.channels), msgs, len(c.ws.Registry.Agents), calls)
	if errs > 0 {
		s += fmt.Sprintf(" (%d err)", errs)
	}
	return s
}

func (c *chat) helpView() string {
	rows := [][2]string{
		{"[ ]   ← →   h l", "chuyển thread (channel)"},
		{"↑ ↓   k j", "cuộn tin nhắn (1 dòng)"},
		{"PgUp/PgDn  b/space  ^u/^d", "cuộn nửa trang"},
		{"g / G   (Home/End)", "lên đầu / xuống tin mới nhất"},
		{"f", "bật/tắt LIVE follow (tự bám tin mới)"},
		{"?", "đóng/mở trợ giúp này"},
		{"q  esc  ^c", "thoát"},
	}
	var b strings.Builder
	b.WriteString(hdr.Render("Hatch chat — phím tắt") + "\n\n")
	for _, r := range rows {
		b.WriteString("  " + selSty.Render(fmt.Sprintf("%-26s", r[0])) + dim.Render(r[1]) + "\n")
	}
	b.WriteString("\n" + dim.Render("Live: tự refresh mỗi giây. Cuộn lên = tạm dừng bám (PAUSED); G hoặc f để bám lại."))
	b.WriteString("\n" + dim.Render("Read-only — agent post qua Hatch MCP (chat_open/chat_post)."))
	return focused.Width(max(40, c.w-4)).Render(b.String())
}
