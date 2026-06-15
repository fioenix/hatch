//go:build hatch_legacy

// Package ceremony runs the recurring squad rituals declared in workflow.yaml:
// standup (per-agent digest + blockers), retro (cycle summary + KB promotion
// candidates). Planning is delegated to the orchestrator (spawn the Conductor).
package ceremony

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/port"
)

// Service runs the squad rituals as a use-case over ports — it depends on no
// concrete infrastructure (the composition root injects adapters).
type Service struct {
	Board  port.Board
	Ledger port.Ledger
	Bus    port.Bus
	KB     port.KB
}

// StandupReport summarises recent activity per agent plus current blockers.
type StandupReport struct {
	Markdown string
	PerAgent map[string][]string
	Blockers []model.Ticket
}

// Standup builds a digest from the last `days` of ledger activity and the
// board's blocked lanes — the deterministic equivalent of "what did you do,
// what's next, what's blocking you".
func (s Service) Standup(ws *config.Workspace, days int) (*StandupReport, error) {
	if days < 1 {
		days = 1
	}
	entries, err := s.Ledger.Recent(days)
	if err != nil {
		return nil, err
	}
	perAgent := map[string][]string{}
	for _, e := range entries {
		if e.Agent == "" {
			continue
		}
		line := e.Action
		if e.Ticket != "" && e.Ticket != "-" {
			line += " " + e.Ticket
		}
		if e.Result != "" {
			line += " (" + e.Result + ")"
		}
		perAgent[e.Agent] = appendUnique(perAgent[e.Agent], line)
	}

	blockers, err := s.blockedTickets(ws)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("# Standup\n\n")
	if len(perAgent) == 0 {
		b.WriteString("_Không có hoạt động ledger gần đây._\n")
	}
	for _, agent := range sortedKeys(perAgent) {
		fmt.Fprintf(&b, "**%s**: %s\n", agent, strings.Join(perAgent[agent], ", "))
	}
	if len(blockers) > 0 {
		b.WriteString("\n## Blockers\n")
		for _, t := range blockers {
			fmt.Fprintf(&b, "- %s (%s) — %s\n", t.ID, t.Lane, t.Title)
		}
	}
	return &StandupReport{Markdown: b.String(), PerAgent: perAgent, Blockers: blockers}, nil
}

// RetroReport summarises a cycle and lists KB promotion candidates.
type RetroReport struct {
	Markdown       string
	Done           int
	GateFailures   int
	Blocks         int
	Decisions      int
	PromotionCands []model.KBEntry
}

// Retro summarises the whole ledger plus bus decisions and surfaces KB
// learnings as SSOT-promotion candidates.
func (s Service) Retro(ws *config.Workspace) (*RetroReport, error) {
	entries, err := s.Ledger.Recent(0)
	if err != nil {
		return nil, err
	}
	r := &RetroReport{}
	for _, e := range entries {
		switch e.Action {
		case model.ActDone:
			r.Done++
		case model.ActBlock:
			r.Blocks++
		case model.ActGate:
			if strings.HasPrefix(e.Result, "failed") {
				r.GateFailures++
			}
		}
	}
	// Decisions recorded on the bus.
	b := s.Bus
	if decs, err := b.Search(model.SearchOpts{Type: model.MsgDecision, Limit: 100}); err == nil {
		r.Decisions = len(decs)
	}
	// KB learnings are candidates to promote into SSOT.
	if entriesKB, err := s.KB.List(); err == nil {
		for _, e := range entriesKB {
			if e.Type == model.KBLearning {
				r.PromotionCands = append(r.PromotionCands, e)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("# Retro\n\n")
	fmt.Fprintf(&sb, "- Done: %d\n- Blocks: %d\n- Gate failures: %d\n- Decisions: %d\n", r.Done, r.Blocks, r.GateFailures, r.Decisions)
	if len(r.PromotionCands) > 0 {
		sb.WriteString("\n## Ứng viên đề bạt KB → SSOT (learnings)\n")
		for _, e := range r.PromotionCands {
			fmt.Fprintf(&sb, "- %s %s — `kb/%s`\n", e.ID, e.Title, e.Path)
		}
		sb.WriteString("\nĐề bạt thủ công sau review (Architect/Conductor).\n")
	}
	r.Markdown = sb.String()
	return r, nil
}

// Demo lists work in terminal (done-like) lanes — the showcase for a sprint
// review / demo. Returns a report and the tickets shown.
func (s Service) Demo(ws *config.Workspace) (string, []model.Ticket, error) {
	var done []model.Ticket
	for _, id := range terminalLaneIDs(ws) {
		ts, err := s.Board.ListLane(id)
		if err != nil {
			return "", nil, err
		}
		done = append(done, ts...)
	}
	var b strings.Builder
	b.WriteString("# Demo / Sprint review\n\n")
	if len(done) == 0 {
		b.WriteString("_Chưa có việc hoàn thành để trình diễn._\n")
	}
	for _, t := range done {
		who := t.Assignee
		if who == "" {
			who = "-"
		}
		fmt.Fprintf(&b, "- **%s** %s — done by %s", t.ID, t.Title, who)
		if t.Epic != "" {
			fmt.Fprintf(&b, " (epic %s)", t.Epic)
		}
		b.WriteString("\n")
	}
	return b.String(), done, nil
}

// GroomItem is a backlog ticket needing refinement plus the reasons.
type GroomItem struct {
	Ticket  model.Ticket
	Reasons []string
}

// Grooming scans the entry (backlog) lane for under-specified tickets — the
// backlog refinement ritual: flag missing role/priority/acceptance.
func (s Service) Grooming(ws *config.Workspace) (string, []GroomItem, error) {
	lane := entryLane(ws)
	tickets, err := s.Board.ListLane(lane)
	if err != nil {
		return "", nil, err
	}
	var items []GroomItem
	for _, t := range tickets {
		var reasons []string
		if t.Role == "" {
			reasons = append(reasons, "thiếu role")
		}
		if t.Priority == "" {
			reasons = append(reasons, "thiếu priority")
		}
		if !strings.Contains(t.Body, "- [ ]") {
			reasons = append(reasons, "thiếu acceptance criteria")
		}
		if strings.Contains(t.Body, "TODO") {
			reasons = append(reasons, "còn TODO trong mô tả")
		}
		if len(reasons) > 0 {
			items = append(items, GroomItem{Ticket: t, Reasons: reasons})
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Backlog grooming (%s)\n\n", lane)
	if len(items) == 0 {
		b.WriteString("_Backlog đã đủ rõ — không có ticket cần chuốt._\n")
	}
	for _, it := range items {
		fmt.Fprintf(&b, "- **%s** %s — %s\n", it.Ticket.ID, it.Ticket.Title, strings.Join(it.Reasons, ", "))
	}
	return b.String(), items, nil
}

func entryLane(ws *config.Workspace) string {
	for _, l := range ws.Workflow.Lanes {
		if !l.Side {
			return l.ID
		}
	}
	return ws.Workflow.Lanes[0].ID
}

func terminalLaneIDs(ws *config.Workspace) []string {
	outgoing := map[string]bool{}
	for _, tr := range ws.Workflow.Transitions {
		if tr.From != "*" {
			outgoing[tr.From] = true
		}
	}
	var out []string
	for _, l := range ws.Workflow.Lanes {
		if !l.Side && !outgoing[l.ID] {
			out = append(out, l.ID)
		}
	}
	return out
}

func (s Service) blockedTickets(ws *config.Workspace) ([]model.Ticket, error) {
	var out []model.Ticket
	for _, l := range ws.Workflow.Lanes {
		if !l.Side && l.ID != "blocked" {
			continue
		}
		ts, err := s.Board.ListLane(l.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, ts...)
	}
	return out, nil
}

func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
