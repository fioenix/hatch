//go:build hatch_legacy

// Package metrics derives operational stats (throughput, cycle time, rework,
// cost…) from the append-only ledger. Everything is computed, never tracked
// separately. See docs/13-management.md (incl. the Goodhart caveat).
package metrics

import (
	"sort"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/port"
)

// AgentStat is one agent's operational scorecard.
type AgentStat struct {
	Agent       string
	Claims      int
	Done        int
	Reviews     int
	GateFails   int
	Escalations int
	CostUSD     float64
	Tokens      int
}

// Report bundles per-agent stats and team-level aggregates.
type Report struct {
	Agents     map[string]*AgentStat
	Throughput int           // total done
	CycleAvg   time.Duration // avg claim→done across tickets
}

// Compute scans the ledger into a Report.
func Compute(lg port.Ledger) (*Report, error) {
	entries, err := lg.Entries()
	if err != nil {
		return nil, err
	}
	r := &Report{Agents: map[string]*AgentStat{}}
	stat := func(a string) *AgentStat {
		if r.Agents[a] == nil {
			r.Agents[a] = &AgentStat{Agent: a}
		}
		return r.Agents[a]
	}

	// First claim + done time per ticket → cycle time.
	firstClaim := map[string]time.Time{}
	var cycles []time.Duration

	for _, e := range entries {
		s := stat(e.Agent)
		s.CostUSD += e.CostUSD
		s.Tokens += e.Tokens
		ts, _ := time.Parse(time.RFC3339, e.TS)
		switch e.Action {
		case model.ActClaim:
			s.Claims++
			if _, ok := firstClaim[e.Ticket]; !ok && !ts.IsZero() {
				firstClaim[e.Ticket] = ts
			}
		case model.ActDone:
			s.Done++
			r.Throughput++
			if c, ok := firstClaim[e.Ticket]; ok && !ts.IsZero() {
				cycles = append(cycles, ts.Sub(c))
			}
		case model.ActReview:
			s.Reviews++
		case model.ActGate:
			if len(e.Result) >= 6 && e.Result[:6] == "failed" {
				s.GateFails++
			}
		case model.ActEscalate:
			s.Escalations++
		}
	}
	if len(cycles) > 0 {
		var total time.Duration
		for _, c := range cycles {
			total += c
		}
		r.CycleAvg = total / time.Duration(len(cycles))
	}
	return r, nil
}

// Sorted returns agent stats ordered by id for stable output.
func (r *Report) Sorted() []*AgentStat {
	out := make([]*AgentStat, 0, len(r.Agents))
	for _, s := range r.Agents {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Agent < out[j].Agent })
	return out
}
