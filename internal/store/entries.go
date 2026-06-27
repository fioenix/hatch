package store

import (
	"os"
	"strconv"
	"strings"

	"github.com/fioenix/hatch/internal/model"
)

// Entries parses every ledger file into structured entries (chronological).
// Powers metrics (workload/perf) and cost reporting.
func (lg *Ledger) Entries() ([]model.Entry, error) {
	files, err := lg.Files()
	if err != nil {
		return nil, err
	}
	var out []model.Entry
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var cur *model.Entry
		flush := func() {
			if cur != nil {
				out = append(out, *cur)
			}
			cur = nil
		}
		for _, ln := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
			if strings.HasPrefix(ln, "## ") {
				flush()
				parts := strings.Split(strings.TrimPrefix(ln, "## "), " · ")
				e := model.Entry{}
				if len(parts) >= 1 {
					e.TS = strings.TrimSpace(parts[0])
				}
				if len(parts) >= 2 {
					e.Agent = strings.TrimSpace(parts[1])
				}
				if len(parts) >= 3 {
					e.Ticket = strings.TrimSpace(parts[2])
				}
				cur = &e
				continue
			}
			if cur == nil {
				continue
			}
			if v, ok := fieldVal(ln, "- action:"); ok {
				cur.Action = v
			} else if v, ok := fieldVal(ln, "- from:"); ok {
				cur.From = v
			} else if v, ok := fieldVal(ln, "- result:"); ok {
				cur.Result = v
			} else if v, ok := fieldVal(ln, "- why:"); ok {
				cur.Why = v
			} else if v, ok := fieldVal(ln, "- handoff:"); ok {
				cur.Handoff = v
			} else if v, ok := fieldVal(ln, "- branch:"); ok {
				cur.Branch = v
			} else if v, ok := fieldVal(ln, "- cost_usd:"); ok {
				cur.CostUSD, _ = strconv.ParseFloat(v, 64)
			} else if v, ok := fieldVal(ln, "- tokens:"); ok {
				cur.Tokens, _ = strconv.Atoi(v)
			}
		}
		flush()
	}
	return out, nil
}
