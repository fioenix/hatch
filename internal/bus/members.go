package bus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// membersPath stores channel → subscribed agents.
func (b *Bus) membersPath() string { return filepath.Join(b.dir(), ".members.json") }

func (b *Bus) loadMembers() map[string][]string {
	raw, err := os.ReadFile(b.membersPath())
	if err != nil {
		return map[string][]string{}
	}
	m := map[string][]string{}
	if json.Unmarshal(raw, &m) != nil {
		return map[string][]string{}
	}
	return m
}

func (b *Bus) saveMembers(m map[string][]string) error {
	if err := os.MkdirAll(b.dir(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.membersPath(), append(raw, '\n'), 0o644)
}

// Subscribe adds an agent to a channel's membership.
func (b *Bus) Subscribe(channel, agent string) error {
	m := b.loadMembers()
	if !contains(m[channel], agent) {
		m[channel] = append(m[channel], agent)
		sort.Strings(m[channel])
	}
	return b.saveMembers(m)
}

// Unsubscribe removes an agent from a channel.
func (b *Bus) Unsubscribe(channel, agent string) error {
	m := b.loadMembers()
	var kept []string
	for _, a := range m[channel] {
		if a != agent {
			kept = append(kept, a)
		}
	}
	m[channel] = kept
	return b.saveMembers(m)
}

// Members lists agents subscribed to a channel.
func (b *Bus) Members(channel string) []string {
	return b.loadMembers()[channel]
}

// Subscriptions lists channels an agent is subscribed to.
func (b *Bus) Subscriptions(agent string) []string {
	var out []string
	for ch, members := range b.loadMembers() {
		if contains(members, agent) {
			out = append(out, ch)
		}
	}
	sort.Strings(out)
	return out
}
