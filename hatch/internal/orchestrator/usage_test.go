//go:build hatch_legacy

package orchestrator

import "testing"

func TestExtractUsage(t *testing.T) {
	usd, tok := extractUsage(`{"result":"ok","total_cost_usd":0.42,"total_tokens":18450}`, 0)
	if usd != 0.42 || tok != 18450 {
		t.Fatalf("parse failed: usd=%v tok=%d", usd, tok)
	}
	// tokens only → estimate from rate (15 USD / 1M tok).
	usd2, _ := extractUsage(`{"tokens":1000000}`, 15)
	if usd2 != 15 {
		t.Fatalf("estimate wrong: %v", usd2)
	}
	if usd3, tok3 := extractUsage("no usage here", 10); usd3 != 0 || tok3 != 0 {
		t.Fatalf("expected zero usage, got %v/%d", usd3, tok3)
	}
}
