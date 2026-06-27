package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fioenix/hatch/internal/model"
	"github.com/fioenix/hatch/internal/paths"
)

// ledgerMu serializes ledger appends within a process so concurrent runs
// (parallel watch/tick) don't interleave entry blocks.
var ledgerMu sync.Mutex

// Ledger appends audit entries to per-day Markdown files.
type Ledger struct{ L paths.Layout }

// NewLedger returns a ledger bound to a workspace layout.
func NewLedger(l paths.Layout) *Ledger { return &Ledger{L: l} }

// render formats an entry as the Markdown block defined by spec/ledger.schema.md.
func render(e model.Entry) string {
	ticket := e.Ticket
	if ticket == "" {
		ticket = "-"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s · %s · %s\n", e.TS, e.Agent, ticket)
	fmt.Fprintf(&b, "- action: %s\n", e.Action)
	if e.From != "" {
		fmt.Fprintf(&b, "- from: %s\n", e.From)
	}
	if e.ToRole != "" {
		fmt.Fprintf(&b, "- to-role: %s\n", e.ToRole)
	}
	if e.Result != "" {
		fmt.Fprintf(&b, "- result: %s\n", e.Result)
	}
	fmt.Fprintf(&b, "- why: %s\n", e.Why)
	if e.Branch != "" {
		fmt.Fprintf(&b, "- branch: %s\n", e.Branch)
	}
	if e.CostUSD > 0 {
		fmt.Fprintf(&b, "- cost_usd: %.4f\n", e.CostUSD)
	}
	if e.Tokens > 0 {
		fmt.Fprintf(&b, "- tokens: %d\n", e.Tokens)
	}
	if e.Handoff != "" {
		fmt.Fprintf(&b, "- handoff: %s\n", e.Handoff)
	}
	if e.Note != "" {
		fmt.Fprintf(&b, "- note: %s\n", e.Note)
	}
	return b.String()
}

// dayFile returns the ledger file path for an entry's timestamp.
func (lg *Ledger) dayFile(ts string) string {
	day := time.Now().Format("2006-01-02")
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		day = t.Format("2006-01-02")
	}
	return filepath.Join(lg.L.Ledger(), day+".md")
}

// Append writes an entry, defaulting its timestamp to now if unset.
func (lg *Ledger) Append(e model.Entry) error {
	if e.TS == "" {
		e.TS = time.Now().Format(time.RFC3339)
	}
	if e.Why == "" {
		return fmt.Errorf("ledger entry requires a non-empty `why`")
	}
	if e.Action == model.ActHandoff && e.Handoff == "" {
		return fmt.Errorf("handoff entry requires a `handoff` note")
	}
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	if err := os.MkdirAll(lg.L.Ledger(), 0o755); err != nil {
		return err
	}
	path := lg.dayFile(e.TS)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	block := render(e)
	if fi, _ := f.Stat(); fi != nil && fi.Size() > 0 {
		block = "\n" + block
	}
	_, err = f.WriteString(block)
	return err
}

// Files lists ledger day-files in chronological order.
func (lg *Ledger) Files() ([]string, error) {
	ents, err := os.ReadDir(lg.L.Ledger())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, filepath.Join(lg.L.Ledger(), e.Name()))
		}
	}
	return out, nil // ReadDir returns sorted names; date format sorts chronologically
}
