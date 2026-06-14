package bus

import (
	"sort"
	"strings"
)

// SearchOpts filters a bus query. Empty fields are ignored. This is the
// token-efficient way an agent loads *relevant* conversation into context
// (L2 on-demand) instead of reading every message in every channel.
type SearchOpts struct {
	Query    string   // case-insensitive substring over body + sender
	Channel  string   // restrict to one channel
	From     string   // restrict to a sender
	Type     string   // restrict to a message type
	Channels []string // restrict to a set of channels (e.g. an agent's subscriptions)
	Limit    int      // max results, newest first (0 ⇒ 20)
}

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
	q := strings.ToLower(o.Query)

	var hits []Message
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
			if q != "" && !strings.Contains(strings.ToLower(m.Body), q) && !strings.Contains(strings.ToLower(m.From), q) {
				continue
			}
			hits = append(hits, m)
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].TS > hits[j].TS })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}
