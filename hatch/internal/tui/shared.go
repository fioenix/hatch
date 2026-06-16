// Package tui renders read-only dashboards with Bubble Tea. The single view is
// `hatch chat` (chat.go): a live, Slack-style view of the squad's shared chat
// with squad stats in the footer. `hatch board` is an alias for it. It only
// observes — agents act through the Hatch MCP server.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	hdr     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2563eb"))
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selSty  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7c3aed"))
	laneSty = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5e9"))
	focused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7c3aed"))
	blurred = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// trunc shortens s to n runes with an ellipsis.
func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// paneBox draws a titled, bordered pane sized to w×h (0 = auto).
func paneBox(focus bool, title, content string, w, h int) string {
	style := blurred
	if focus {
		style = focused
	}
	if w > 0 {
		style = style.Width(w)
	}
	if h > 0 {
		style = style.Height(h)
	}
	return style.Render(laneSty.Render(title) + "\n" + content)
}
