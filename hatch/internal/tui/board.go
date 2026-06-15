// Package tui renders read-only dashboards with Bubble Tea. `hatch board` is
// mission control: THREADS (chat channels — each thread is a task) + CHAT (the
// selected thread) + ACTIVITY (the ledger), in one view. It only observes; it
// never drives agents. `hatch chat` (chat.go) is a focused stand-alone chat
// view sharing the same widgets. Agents act through the Hatch MCP server.
package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

type pane int

const (
	paneThreads pane = iota
	paneChat
	paneActivity
	numPanes
)

var (
	hdr     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2563eb"))
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selSty  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7c3aed"))
	laneSty = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5e9"))
	focused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7c3aed"))
	blurred = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
)

// threadStat summarises one chat thread (= one task) for the THREADS pane.
type threadStat struct {
	name  string
	count int
	last  string // HH:MM of the last message
}

type m struct {
	ws     *config.Workspace
	bus    *bus.Bus
	ledger *store.Ledger

	threads  []threadStat
	sel      int
	focus    pane
	chat     viewport.Model
	activity viewport.Model

	w, h   int
	status string
	ready  bool
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// New returns the read-only mission-control dashboard program.
func New(ws *config.Workspace) *tea.Program {
	mm := m{
		ws:     ws,
		bus:    bus.New(ws.Layout),
		ledger: store.NewLedger(ws.Layout),
	}
	return tea.NewProgram(&mm, tea.WithAltScreen())
}

func (mm *m) Init() tea.Cmd { return tick() }

func (mm *m) reload() {
	chs, _ := mm.bus.Channels()
	sort.Strings(chs)
	stats := make([]threadStat, 0, len(chs))
	for _, ch := range chs {
		msgs, _ := mm.bus.Messages(ch)
		last := ""
		if len(msgs) > 0 {
			if t, e := time.Parse(time.RFC3339Nano, msgs[len(msgs)-1].TS); e == nil {
				last = t.Format("15:04")
			}
		}
		stats = append(stats, threadStat{name: ch, count: len(msgs), last: last})
	}
	mm.threads = stats
	if mm.sel >= len(stats) {
		mm.sel = max(0, len(stats)-1)
	}
	mm.chat.SetContent(mm.chatFeed())
	mm.activity.SetContent(mm.activityFeed())
}

func (mm *m) curChannel() string {
	if mm.sel < len(mm.threads) {
		return mm.threads[mm.sel].name
	}
	return ""
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

func (mm *m) chatFeed() string {
	ch := mm.curChannel()
	if ch == "" {
		return dim.Render("(chưa có thread nào)")
	}
	msgs, err := mm.bus.Messages(ch)
	if err != nil || len(msgs) == 0 {
		return dim.Render("(thread trống)")
	}
	var b strings.Builder
	for _, msg := range msgs {
		ts := msg.TS
		if t, e := time.Parse(time.RFC3339Nano, msg.TS); e == nil {
			ts = t.Format("15:04")
		}
		head := fmt.Sprintf("%s %s", dim.Render(ts), selSty.Render(msg.From))
		if msg.Type != bus.TypeMsg {
			head += " " + laneSty.Render("["+msg.Type+"]")
		}
		b.WriteString(head + "\n  " + strings.ReplaceAll(msg.Body, "\n", "\n  ") + "\n")
	}
	return b.String()
}

func (mm *m) layout() {
	rightW := mm.w - mm.w/2 - 4
	vpH := (mm.h - 6) / 2
	if vpH < 3 {
		vpH = 3
	}
	mm.chat = viewport.New(rightW, vpH)
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
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return mm, tea.Quit
		case "tab":
			mm.focus = (mm.focus + 1) % numPanes
		case "up", "k":
			mm.scrollUp()
		case "down", "j":
			mm.scrollDown()
		case "g":
			mm.reload()
		}
	}
	return mm, nil
}

func (mm *m) scrollUp() {
	switch mm.focus {
	case paneThreads:
		if mm.sel > 0 {
			mm.sel--
			mm.chat.SetContent(mm.chatFeed())
		}
	case paneChat:
		mm.chat.LineUp(1)
	case paneActivity:
		mm.activity.LineUp(1)
	}
}

func (mm *m) scrollDown() {
	switch mm.focus {
	case paneThreads:
		if mm.sel < len(mm.threads)-1 {
			mm.sel++
			mm.chat.SetContent(mm.chatFeed())
		}
	case paneChat:
		mm.chat.LineDown(1)
	case paneActivity:
		mm.activity.LineDown(1)
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
		dim.Render(fmt.Sprintf("(read-only · %d threads)", len(mm.threads)))

	// THREADS pane: each chat thread is a task.
	var th strings.Builder
	if len(mm.threads) == 0 {
		th.WriteString(dim.Render("(chưa có thread — agent mở qua Hatch MCP)") + "\n")
	}
	for i, t := range mm.threads {
		line := fmt.Sprintf("  #%-20s %3d  %s", trunc(t.name, 20), t.count, t.last)
		if i == mm.sel {
			line = selSty.Render("▸ #" + trunc(t.name, 20) + fmt.Sprintf("  %d  %s", t.count, t.last))
		}
		th.WriteString(line + "\n")
	}
	threadsBox := paneBox(mm.focus == paneThreads, "THREADS (tasks)", th.String(), mm.w/2-2, mm.h-4)

	chatBox := paneBox(mm.focus == paneChat, "CHAT · #"+mm.curChannel(), mm.chat.View(), 0, 0)
	actBox := paneBox(mm.focus == paneActivity, "ACTIVITY (ledger)", mm.activity.View(), 0, 0)
	right := lipgloss.JoinVertical(lipgloss.Left, chatBox, actBox)
	body := lipgloss.JoinHorizontal(lipgloss.Top, threadsBox, right)

	foot := dim.Render("tab pane · ↑/↓ move·scroll · g refresh · q quit")
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
