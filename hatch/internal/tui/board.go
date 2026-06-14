// Package tui renders an interactive board dashboard with Bubble Tea.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fcaf16"))
	laneStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2a2b86")).
			BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	overWIP   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d33"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle = dimStyle.Copy().MarginTop(1)
)

type model_ struct {
	ws   *config.Workspace
	b    *store.Board
	err  error
	cols []laneCol
}

type laneCol struct {
	lane    model.Lane
	tickets []model.Ticket
}

// New returns a Bubble Tea program for the board.
func New(ws *config.Workspace) *tea.Program {
	m := model_{ws: ws, b: store.NewBoard(ws.Layout)}
	m.reload()
	return tea.NewProgram(m, tea.WithAltScreen())
}

func (m *model_) reload() {
	m.cols = nil
	for _, lane := range m.ws.Workflow.Lanes {
		ts, err := m.b.ListLane(lane.ID)
		if err != nil {
			m.err = err
			return
		}
		m.cols = append(m.cols, laneCol{lane: lane, tickets: ts})
	}
}

func (m model_) Init() tea.Cmd { return nil }

func (m model_) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.reload()
		}
	}
	return m, nil
}

func (m model_) View() string {
	if m.err != nil {
		return "error: " + m.err.Error() + "\n"
	}
	var b strings.Builder
	project := m.ws.Registry.Project
	if project == "" {
		project = "Hatch"
	}
	b.WriteString(titleStyle.Render(project+" — board") + "  " + dimStyle.Render("("+m.ws.Workflow.Template+")") + "\n\n")

	for _, c := range m.cols {
		head := fmt.Sprintf("%s  %d", c.lane.ID, len(c.tickets))
		if c.lane.WIPLimit > 0 {
			wip := fmt.Sprintf(" (WIP %d/%d)", len(c.tickets), c.lane.WIPLimit)
			if len(c.tickets) > c.lane.WIPLimit {
				head += overWIP.Render(wip)
			} else {
				head += dimStyle.Render(wip)
			}
		}
		b.WriteString(laneStyle.Render(head) + "\n")
		if len(c.tickets) == 0 {
			b.WriteString(dimStyle.Render("  —") + "\n")
		}
		for _, t := range c.tickets {
			assignee := t.Assignee
			if assignee == "" {
				assignee = "-"
			}
			prio := t.Priority
			if prio == "" {
				prio = "--"
			}
			line := fmt.Sprintf("  %-7s %-3s %-12s %-12s %s", t.ID, prio, t.Role, assignee, t.Title)
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("r reload · q quit"))
	return b.String()
}
