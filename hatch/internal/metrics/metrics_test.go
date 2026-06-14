package metrics

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func TestComputeFromLedger(t *testing.T) {
	l := paths.At(t.TempDir())
	lg := store.NewLedger(l)
	lg.Append(model.Entry{TS: "2026-06-14T09:00:00Z", Agent: "codex", Ticket: "T-1", Action: model.ActClaim, Why: "x"})
	lg.Append(model.Entry{Agent: "codex", Ticket: "T-1", Action: model.ActGate, Result: "failed: test", Why: "x"})
	lg.Append(model.Entry{TS: "2026-06-14T10:00:00Z", Agent: "claude-code", Ticket: "T-1", Action: model.ActDone, Why: "x", CostUSD: 0.5})

	rep, err := Compute(lg)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Throughput != 1 {
		t.Fatalf("throughput = %d, want 1", rep.Throughput)
	}
	if rep.Agents["codex"].Claims != 1 || rep.Agents["codex"].GateFails != 1 {
		t.Fatalf("codex stat wrong: %+v", rep.Agents["codex"])
	}
	if rep.Agents["claude-code"].Done != 1 || rep.Agents["claude-code"].CostUSD != 0.5 {
		t.Fatalf("claude stat wrong: %+v", rep.Agents["claude-code"])
	}
	if rep.CycleAvg.Hours() != 1 {
		t.Fatalf("cycle avg = %s, want 1h", rep.CycleAvg)
	}
}
