package bus

import (
	"strings"
)

// parseThread parses the append-only Markdown of a thread back into messages.
// Heading form: "## <ts> · <from> → <to> · <type>[ · re:<id>] · {#<id>}".
func parseThread(channel, raw string) []Message {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	var msgs []Message
	var cur *Message
	var body []string

	flush := func() {
		if cur != nil {
			cur.Body = strings.TrimSpace(strings.Join(body, "\n"))
			msgs = append(msgs, *cur)
		}
		cur = nil
		body = nil
	}

	for _, ln := range lines {
		if strings.HasPrefix(ln, "## ") {
			flush()
			m := parseHeading(strings.TrimPrefix(ln, "## "))
			m.Channel = channel
			cur = &m
			continue
		}
		if cur != nil {
			body = append(body, ln)
		}
	}
	flush()
	return msgs
}

func parseHeading(h string) Message {
	var m Message
	// trailing {#id}
	if i := strings.LastIndex(h, "{#"); i >= 0 {
		if j := strings.Index(h[i:], "}"); j >= 0 {
			m.ID = h[i+2 : i+j]
			h = strings.TrimRight(h[:i], " ·")
		}
	}
	parts := strings.Split(h, " · ")
	for idx, p := range parts {
		p = strings.TrimSpace(p)
		switch {
		case idx == 0:
			m.TS = p
		case strings.Contains(p, "→"):
			fromTo := strings.SplitN(p, "→", 2)
			m.From = strings.TrimSpace(fromTo[0])
			for _, to := range strings.Split(fromTo[1], ",") {
				if t := strings.TrimSpace(to); t != "" {
					m.To = append(m.To, t)
				}
			}
		case strings.HasPrefix(p, "re:"):
			m.InReplyTo = strings.TrimPrefix(p, "re:")
		case p == TypeMsg || p == TypeAsk || p == TypeReply || p == TypeDecision:
			m.Type = p
		}
	}
	return m
}
