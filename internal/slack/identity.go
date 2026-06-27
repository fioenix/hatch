package slack

import "github.com/fioenix/hatch/internal/model"

// identity maps a sender id to how it should appear in Slack: a display name
// and an icon emoji. One bot token posts for everyone; the per-message username
// + icon override is what makes each agent show up as itself.
func identity(r model.Roster, from string) (name, icon string) {
	name = from
	kind := from // fall back to id when the sender isn't a roster member
	if m, ok := r[from]; ok {
		if m.Note != "" {
			name = m.Note
		}
		kind = m.Kind
	}
	return name, iconFor(kind)
}

// iconFor returns a stable emoji per agent kind so the room is readable at a
// glance. "hatch" is the bridge/daemon voice (escalations).
func iconFor(kind string) string {
	switch kind {
	case "claude":
		return ":robot_face:"
	case "codex":
		return ":gear:"
	case "agy":
		return ":sparkles:"
	case "kiro":
		return ":owl:"
	case model.KindUser:
		return ":bust_in_silhouette:"
	case "hatch":
		return ":bell:"
	default:
		return ":speech_balloon:"
	}
}
