package bus

import (
	"sort"
	"strings"

	"github.com/fioenix/hatch/internal/model"
)

// SearchOpts is the domain query type, re-exported for terse call sites.
type SearchOpts = model.SearchOpts

// Search returns matching messages newest-first, capped at Limit.
func (b *Bus) Search(o SearchOpts) ([]Message, error) {
	limit := o.Limit
	if limit <= 0 {
		limit = 20
	}
	scope := o.Channels
	if o.Channel != "" {
		scope = []string{o.Channel}
	}
	if len(scope) == 0 {
		chs, err := b.Channels()
		if err != nil {
			return nil, err
		}
		scope = chs
	}
	allow := map[string]bool{}
	for _, c := range scope {
		allow[safeThread(c)] = true
	}
	tokens := queryTokens(o.Query)

	type scored struct {
		m     Message
		score int
	}
	var hits []scored
	channels, err := b.Channels()
	if err != nil {
		return nil, err
	}
	for _, ch := range channels {
		if !allow[ch] {
			continue
		}
		msgs, err := b.Messages(ch)
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			if o.From != "" && m.From != o.From {
				continue
			}
			if o.Type != "" && m.Type != o.Type {
				continue
			}
			score := matchScore(tokens, m)
			if len(tokens) > 0 && score == 0 {
				continue
			}
			hits = append(hits, scored{m, score})
		}
	}
	// Rank by number of distinct query tokens matched, then by recency.
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].m.TS > hits[j].m.TS
	})
	out := make([]Message, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.m)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// queryTokens lowercases the query and keeps tokens of length >= 2.
func queryTokens(q string) []string {
	var out []string
	for _, t := range strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= 'à' && r <= 'ỹ')
	}) {
		if len([]rune(t)) >= 2 {
			out = append(out, t)
		}
	}
	return out
}

// matchScore counts how many distinct query tokens appear in body or sender.
func matchScore(tokens []string, m Message) int {
	if len(tokens) == 0 {
		return 0
	}
	hay := strings.ToLower(m.Body + " " + m.From)
	n := 0
	for _, t := range tokens {
		if strings.Contains(hay, t) {
			n++
		}
	}
	return n
}
