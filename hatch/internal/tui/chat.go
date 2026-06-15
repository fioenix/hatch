package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
)

type chatFocus int

const (
	focusChannels chatFocus = iota
	focusMessages
)

type chat struct {
	ws       *config.Workspace
	bus      *bus.Bus
	channels []string
	sel      int
	focus    chatFocus
	msgs     viewport.Model
	w, h     int
	ready    bool
}

// NewChat returns a read-only Slack-style TUI for observing the squad's
// communication bus. Agents post through the Hatch MCP server; this view only
// watches.
func NewChat(ws *config.Workspace) *tea.Program {
	c := &chat{ws: ws, bus: bus.New(ws.Layout)}
	return tea.NewProgram(c, tea.WithAltScreen())
}

func (c *chat) Init() tea.Cmd { return tick() }

func (c *chat) reload() {
	chs, _ := c.bus.Channels()
	sort.Strings(chs)
	c.channels = chs
	if c.sel >= len(chs) {
		c.sel = maxi(0, len(chs)-1)
	}
	c.msgs.SetContent(c.renderMessages())
}

func (c *chat) current() string {
	if c.sel < len(c.channels) {
		return c.channels[c.sel]
	}
	return ""
}

func (c *chat) renderMessages() string {
	ch := c.current()
	if ch == "" {
		return dim.Render("(chưa có thread — agent mở qua Hatch MCP)")
	}
	msgs, err := c.bus.Messages(ch)
	if err != nil || len(msgs) == 0 {
		return dim.Render("(thread trống)")
	}
	var b strings.Builder
	for _, m := range msgs {
		ts := m.TS
		if t, e := time.Parse(time.RFC3339Nano, m.TS); e == nil {
			ts = t.Format("15:04")
		}
		head := fmt.Sprintf("%s %s", dim.Render(ts), selSty.Render(m.From))
		if m.Type != bus.TypeMsg {
			head += " " + laneSty.Render("["+m.Type+"]")
		}
		if len(m.To) > 0 {
			head += dim.Render(" → " + strings.Join(m.To, ","))
		}
		b.WriteString(head + "\n  " + strings.ReplaceAll(m.Body, "\n", "\n  ") + "\n\n")
	}
	return b.String()
}

func (c *chat) layout() {
	mw := c.w - c.w/4 - 4
	c.msgs = viewport.New(mw, c.h-5)
	c.ready = true
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
			c.msgs.GotoBottom()
		}
		return c, tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return c, tea.Quit
		case "tab":
			c.focus = (c.focus + 1) % 2
		case "up", "k":
			if c.focus == focusChannels && c.sel > 0 {
				c.sel--
				c.reload()
			} else {
				c.msgs.LineUp(1)
			}
		case "down", "j":
			if c.focus == focusChannels && c.sel < len(c.channels)-1 {
				c.sel++
				c.reload()
			} else {
				c.msgs.LineDown(1)
			}
		case "g":
			c.reload()
		}
	}
	return c, nil
}

func (c *chat) View() string {
	if !c.ready {
		return "loading…"
	}
	project := c.ws.Registry.Project
	if project == "" {
		project = "Hatch"
	}
	header := hdr.Render(project+" — chat") + "  " + dim.Render("(read-only)")

	// Channel list.
	var cl strings.Builder
	for i, ch := range c.channels {
		line := "  " + ch
		if i == c.sel {
			line = selSty.Render("▸ " + ch)
		}
		cl.WriteString(line + "\n")
	}
	if len(c.channels) == 0 {
		cl.WriteString(dim.Render("  (chưa có)"))
	}
	chanBox := paneBox(c.focus == focusChannels, "CHANNELS", cl.String(), c.w/4, c.h-5)
	msgBox := paneBox(c.focus == focusMessages, "#"+c.current(), c.msgs.View(), 0, 0)
	body := lipgloss.JoinHorizontal(lipgloss.Top, chanBox, msgBox)

	foot := dim.Render("tab pane · ↑/↓ channel·scroll · g refresh · q quit")
	return header + "\n" + body + "\n" + foot
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
