package store

import (
	"os"
	"strings"

	"github.com/fioenix/hatch/internal/model"
)

// parseLedger parses a ledger day-file back into entries. Heading form:
// "## <ts> · <agent> · <ticket>" followed by "- key: value" lines.
func parseLedger(raw string) []model.Entry {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	var entries []model.Entry
	var cur *model.Entry
	flush := func() {
		if cur != nil {
			entries = append(entries, *cur)
		}
		cur = nil
	}
	for _, ln := range strings.Split(raw, "\n") {
		if strings.HasPrefix(ln, "## ") {
			flush()
			e := model.Entry{}
			parts := strings.SplitN(strings.TrimPrefix(ln, "## "), " · ", 3)
			if len(parts) > 0 {
				e.TS = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				e.Agent = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				e.Ticket = strings.TrimSpace(parts[2])
			}
			cur = &e
			continue
		}
		if cur == nil || !strings.HasPrefix(ln, "- ") {
			continue
		}
		kv := strings.SplitN(strings.TrimPrefix(ln, "- "), ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "action":
			cur.Action = val
		case "from":
			cur.From = val
		case "to-role":
			cur.ToRole = val
		case "result":
			cur.Result = val
		case "why":
			cur.Why = val
		case "handoff":
			cur.Handoff = val
		case "branch":
			cur.Branch = val
		case "note":
			cur.Note = val
		}
	}
	flush()
	return entries
}

// Recent returns ledger entries from the most recent `days` day-files,
// oldest-first.
func (lg *Ledger) Recent(days int) ([]model.Entry, error) {
	files, err := lg.Files()
	if err != nil {
		return nil, err
	}
	if days > 0 && len(files) > days {
		files = files[len(files)-days:]
	}
	var out []model.Entry
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		out = append(out, parseLedger(string(raw))...)
	}
	return out, nil
}
