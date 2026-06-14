// Package tui renders an interactive mission-control dashboard with Bubble Tea:
// the board, live agent output (run transcripts) and an activity feed in one
// multi-pane view, plus the ability to launch a run on the selected ticket.
package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/orchestrator"
	"github.com/fioenix/overclaud/hatch/internal/presence"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

type pane int

const (
	paneBoard pane = iota
	paneLive
	paneActivity
)

var (
	hdr     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2563eb"))
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selSty  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7c3aed"))
	laneSty = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5e9"))
	focused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7c3aed"))
	blurred = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
)

type tref struct {
	lane string
	t    model.Ticket
}

type m struct {
	ws       *config.Workspace
	board    *store.Board
	ledger   *store.Ledger
	refs     []tref
	sel      int
	focus    pane
	live     viewport.Model
	activity viewport.Model
	liveTick string
	w, h     int
	status   string
	ready    bool
}

type tickMsg time.Time
type ranMsg struct {
	ticket string
	err    error
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// New returns a Bubble Tea program for the mission-control dashboard.
func New(ws *config.Workspace) *tea.Program {
	mm := m{ws: ws, board: store.NewBoard(ws.Layout), ledger: store.NewLedger(ws.Layout)}
	return tea.NewProgram(&mm, tea.WithAltScreen())
}

func (mm *m) Init() tea.Cmd { return tick() }

func (mm *m) reload() {
	var refs []tref
	for _, lane := range mm.ws.Workflow.Lanes {
		ts, _ := mm.board.ListLane(lane.ID)
		for _, t := range ts {
			refs = append(refs, tref{lane.ID, t})
		}
	}
	mm.refs = refs
	if mm.sel >= len(refs) {
		mm.sel = max(0, len(refs)-1)
	}
	if mm.liveTick == "" && len(refs) > 0 {
		mm.liveTick = refs[mm.sel].t.ID
	}
	mm.live.SetContent(mm.transcript(mm.liveTick))
	mm.activity.SetContent(mm.activityFeed())
}

func (mm *m) transcript(ticket string) string {
	if ticket == "" {
		return dim.Render("(chọn ticket, nhấn f để theo dõi live output)")
	}
	dir := mm.ws.Layout.Runs(ticket)
	ents, err := os.ReadDir(dir)
	if err != nil || len(ents) == 0 {
		return dim.Render("(chưa có run nào cho " + ticket + ")")
	}
	var logs []string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".log") {
			logs = append(logs, e.Name())
		}
	}
	if len(logs) == 0 {
		return dim.Render("(chưa có transcript)")
	}
	sort.Strings(logs)
	raw, _ := os.ReadFile(filepath.Join(dir, logs[len(logs)-1]))
	return string(raw)
}

func (mm *m) activityFeed() string {
	files, _ := mm.ledger.Files()
	if len(files) == 0 {
		return dim.Render("(ledger trống)")
	}
	raw, _ := os.ReadFile(files[len(files)-1])
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	return strings.Join(lines, "\n")
}

func (mm *m) layout() {
	leftW := mm.w/2 - 2
	rightW := mm.w - leftW - 4
	vpH := (mm.h - 6) / 2
	mm.live = viewport.New(rightW, vpH)
	mm.activity = viewport.New(rightW, vpH)
	mm.ready = true
}

func (mm *m) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		mm.w, mm.h = msg.Width, msg.Height
		mm.layout()
		mm.reload()
	case tickMsg:
		if mm.ready {
			mm.reload()
		}
		return mm, tick()
	case ranMsg:
		if msg.err != nil {
			mm.status = "run " + msg.ticket + " lỗi: " + msg.err.Error()
		} else {
			mm.status = "run " + msg.ticket + " xong"
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return mm, tea.Quit
		case "tab":
			mm.focus = (mm.focus + 1) % 3
		case "up", "k":
			if mm.focus == paneBoard && mm.sel > 0 {
				mm.sel--
			} else if mm.focus == paneLive {
				mm.live.LineUp(1)
			} else if mm.focus == paneActivity {
				mm.activity.LineUp(1)
			}
		case "down", "j":
			if mm.focus == paneBoard && mm.sel < len(mm.refs)-1 {
				mm.sel++
			} else if mm.focus == paneLive {
				mm.live.LineDown(1)
			} else if mm.focus == paneActivity {
				mm.activity.LineDown(1)
			}
		case "f":
			if mm.focus == paneBoard && mm.sel < len(mm.refs) {
				mm.liveTick = mm.refs[mm.sel].t.ID
				mm.status = "theo dõi " + mm.liveTick
			}
		case "r":
			if mm.sel < len(mm.refs) {
				return mm, mm.runSelected()
			}
		case "g":
			mm.reload()
		}
	}
	return mm, nil
}

// runSelected launches an agent on the selected ticket in the background.
func (mm *m) runSelected() tea.Cmd {
	ref := mm.refs[mm.sel]
	mm.liveTick = ref.t.ID
	agent, ok := pick(mm.ws, ref.t.Role)
	if !ok {
		mm.status = "không có agent rảnh cho vai " + ref.t.Role
		return nil
	}
	mm.status = "đang chạy " + agent.ID + " trên " + ref.t.ID + "…"
	ws, t, role := mm.ws, ref.t, ref.t.Role
	return func() tea.Msg {
		_, err := orchestrator.Run(ws, agent, t, role, orchestrator.RunOptions{Stdout: io.Discard})
		return ranMsg{ticket: t.ID, err: err}
	}
}

func (mm *m) View() string {
	if !mm.ready {
		return "loading…"
	}
	project := mm.ws.Registry.Project
	if project == "" {
		project = "Hatch"
	}
	header := hdr.Render(project+" — mission control") + "  " +
		dim.Render(fmt.Sprintf("(%s · %d tickets)", mm.ws.Workflow.Template, len(mm.refs)))

	// Board pane.
	var bd strings.Builder
	curLane := ""
	for i, r := range mm.refs {
		if r.lane != curLane {
			curLane = r.lane
			bd.WriteString(laneSty.Render(curLane) + "\n")
		}
		line := fmt.Sprintf("  %-7s %-11s %s", r.t.ID, r.t.Assignee, trunc(r.t.Title, 22))
		if i == mm.sel {
			line = selSty.Render("▸ " + strings.TrimLeft(line, " "))
		}
		bd.WriteString(line + "\n")
	}
	boardBox := paneBox(mm.focus == paneBoard, "BOARD", bd.String(), mm.w/2-2, mm.h-6)

	mm.live.SetContent(mm.transcript(mm.liveTick))
	liveBox := paneBox(mm.focus == paneLive, "LIVE · "+mm.liveTick, mm.live.View(), 0, 0)
	actBox := paneBox(mm.focus == paneActivity, "ACTIVITY (ledger)", mm.activity.View(), 0, 0)
	right := lipgloss.JoinVertical(lipgloss.Left, liveBox, actBox)
	body := lipgloss.JoinHorizontal(lipgloss.Top, boardBox, right)

	foot := dim.Render("tab pane · ↑/↓ move·scroll · f follow · r run · g refresh · q quit")
	if mm.status != "" {
		foot = selSty.Render(mm.status) + "   " + foot
	}
	return header + "\n" + body + "\n" + foot
}

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

// pick chooses an available agent for a role (presence-aware, first match).
func pick(ws *config.Workspace, role string) (model.Agent, bool) {
	pres := presence.Load(ws.Layout)
	for _, a := range ws.Registry.AgentsForRole(role) {
		if pres.CanTakeWork(a.ID) {
			return a, true
		}
	}
	return model.Agent{}, false
}

func trunc(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
