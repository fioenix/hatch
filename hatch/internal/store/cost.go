package store

import (
	"os"
	"strconv"
	"strings"
)

// CostRecord attributes a cost/token amount to an agent + ticket.
type CostRecord struct {
	Agent  string
	Ticket string
	USD    float64
	Tokens int
}

// ScanCosts parses all ledger files for cost_usd/tokens lines, attributing each
// to the agent + ticket from its entry heading. This is the track-only basis
// for `hatch cost` / `hatch budget`.
func (lg *Ledger) ScanCosts() ([]CostRecord, error) {
	files, err := lg.Files()
	if err != nil {
		return nil, err
	}
	var out []CostRecord
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var cur *CostRecord
		flush := func() {
			if cur != nil && (cur.USD > 0 || cur.Tokens > 0) {
				out = append(out, *cur)
			}
			cur = nil
		}
		for _, ln := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
			if strings.HasPrefix(ln, "## ") {
				flush()
				// "## <ts> · <agent> · <ticket>"
				parts := strings.Split(strings.TrimPrefix(ln, "## "), " · ")
				rec := CostRecord{}
				if len(parts) >= 2 {
					rec.Agent = strings.TrimSpace(parts[1])
				}
				if len(parts) >= 3 {
					rec.Ticket = strings.TrimSpace(parts[2])
				}
				cur = &rec
				continue
			}
			if cur == nil {
				continue
			}
			if v, ok := fieldVal(ln, "- cost_usd:"); ok {
				cur.USD, _ = strconv.ParseFloat(v, 64)
			}
			if v, ok := fieldVal(ln, "- tokens:"); ok {
				cur.Tokens, _ = strconv.Atoi(v)
			}
		}
		flush()
	}
	return out, nil
}

func fieldVal(line, prefix string) (string, bool) {
	if strings.HasPrefix(line, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
	}
	return "", false
}
