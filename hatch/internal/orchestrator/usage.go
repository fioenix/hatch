//go:build hatch_legacy

package orchestrator

import (
	"regexp"
	"strconv"
)

// Usage parsing is best-effort and provider-agnostic: agents print cost/usage
// in their JSON output in different shapes. We scrape what we can; when only
// tokens are known we estimate USD from the agent's configured rate.

var (
	reCostUSD = regexp.MustCompile(`"(?:total_cost_usd|cost_usd|costUSD)"\s*:\s*([0-9.]+)`)
	reTokens  = regexp.MustCompile(`"(?:total_tokens|tokens|token_count|tokens_used)"\s*:\s*([0-9]+)`)
)

// extractUsage scrapes cost (USD) and tokens from an agent's output. If cost is
// absent but tokens and a per-MTok rate are known, it estimates cost.
func extractUsage(output string, ratePerMTok float64) (usd float64, tokens int) {
	if m := reCostUSD.FindStringSubmatch(output); m != nil {
		usd, _ = strconv.ParseFloat(m[1], 64)
	}
	if m := reTokens.FindStringSubmatch(output); m != nil {
		tokens, _ = strconv.Atoi(m[1])
	}
	if usd == 0 && tokens > 0 && ratePerMTok > 0 {
		usd = float64(tokens) / 1_000_000 * ratePerMTok
	}
	return usd, tokens
}
