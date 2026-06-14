// Package ceremony runs the recurring squad rituals declared in workflow.yaml:
// standup (per-agent digest + blockers), retro (cycle summary + KB promotion
// candidates). Planning is delegated to the orchestrator (spawn the Conductor).
package ceremony

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// StandupReport summarises recent activity per agent plus current blockers.
type StandupReport struct {
	Markdown string
	PerAgent map[string][]string
	Blockers []model.Ticket
}

// Standup builds a digest from the last `days` of ledger activity and the
// board's blocked lanes — the deterministic equivalent of "what did you do,
// what's next, what's blocking you".
func Standup(ws *config.Workspace, days int) (*StandupReport, error) {
	if days < 1 {
		days = 1
	}
	entries, err := store.NewLedger(ws.Layout).Recent(days)
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

	blockers, err := blockedTickets(ws)
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
func Retro(ws *config.Workspace) (*RetroReport, error) {
	entries, err := store.NewLedger(ws.Layout).Recent(0)
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
	b := bus.New(ws.Layout)
	if decs, err := b.Search(bus.SearchOpts{Type: bus.TypeDecision, Limit: 100}); err == nil {
		r.Decisions = len(decs)
	}
	// KB learnings are candidates to promote into SSOT.
	if entriesKB, err := store.NewKB(ws.Layout).List(); err == nil {
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

func blockedTickets(ws *config.Workspace) ([]model.Ticket, error) {
	board := store.NewBoard(ws.Layout)
	var out []model.Ticket
	for _, l := range ws.Workflow.Lanes {
		if !l.Side && l.ID != "blocked" {
			continue
		}
		ts, err := board.ListLane(l.ID)
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
