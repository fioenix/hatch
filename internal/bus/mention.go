package bus

import "regexp"

// mentionRe matches @handles: @codex, @claude-code, @reviewer, @all.
var mentionRe = regexp.MustCompile(`@([A-Za-z0-9][A-Za-z0-9._-]*)`)

// Mentions extracts @-tagged handles (agent ids or roles) from a message body,
// without the leading '@'. Used to route tags to recipients' inboxes.
func Mentions(body string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range mentionRe.FindAllStringSubmatch(body, -1) {
		h := m[1]
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	return out
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
