package cli

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/bus"
)

func TestBriefText(t *testing.T) {
	// Nothing waiting and no threads → empty (hook stays silent).
	if got := briefText("codex", nil, nil, nil); got != "" {
		t.Errorf("want empty briefing, got %q", got)
	}

	// Inbox + threads → mentions the agent, the sender, and the open thread.
	msgs := []bus.Message{{Channel: "#export-csv", From: "claude-code", Body: "@codex stream it"}}
	got := briefText("codex", []string{"implementer"}, msgs, []string{"#export-csv"})
	for _, want := range []string{"codex", "implementer", "claude-code", "export-csv", "stream it"} {
		if !strings.Contains(got, want) {
			t.Errorf("briefing missing %q:\n%s", want, got)
		}
	}

	// Open threads but empty inbox → still briefs the threads.
	if got := briefText("kiro", nil, nil, []string{"#a"}); !strings.Contains(got, "#a") {
		t.Errorf("want thread in briefing, got %q", got)
	}
}
