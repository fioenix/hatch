package bus

import (
	"fmt"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/port"
)

// Bus implements port.Bus.
var _ port.Bus = (*Bus)(nil)

// Notify posts a plain message to a channel (the write side of port.Bus).
func (b *Bus) Notify(channel, from string, to []string, body string) error {
	_, err := b.Post(Message{Channel: channel, From: from, To: to, Body: body})
	return err
}

// CatchUp returns an agent's unread inbox and a query-scoped recall of recent
// conversation, formatted as compact, token-bounded lines (the read side an
// agent uses to "read the room" before starting work).
func (b *Bus) CatchUp(agent string, roles []string, query string, limit int) (inbox, recall []string) {
	in, _ := b.Inbox(agent, roles)
	for _, m := range capMsgs(in, 10) {
		inbox = append(inbox, fmt.Sprintf("[%s] %s · %s: %s", m.Type, m.Channel, m.From, snippet(m.Body)))
	}
	subs := b.Subscriptions(agent)
	rc, _ := b.Search(SearchOpts{Query: query, Channels: subs, Limit: limit})
	for _, m := range rc {
		recall = append(recall, fmt.Sprintf("%s · %s: %s", m.Channel, m.From, snippet(m.Body)))
	}
	return inbox, recall
}

func capMsgs(ms []Message, n int) []Message {
	if len(ms) > n {
		return ms[len(ms)-n:]
	}
	return ms
}

func snippet(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len([]rune(s)) > 120 {
		return string([]rune(s)[:120]) + "…"
	}
	return s
}
