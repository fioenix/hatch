package mux

import (
	"strings"
	"testing"
)

func TestCommandTmuxAndZellij(t *testing.T) {
	tm, err := Command(Tmux, "T-1", []string{"hatch", "run", "T-1", "--agent", "codex"})
	if err != nil || tm[0] != "tmux" || tm[1] != "new-window" {
		t.Fatalf("tmux command wrong: %v (%v)", tm, err)
	}
	if !strings.Contains(strings.Join(tm, " "), "hatch run T-1") {
		t.Fatalf("tmux inner missing: %v", tm)
	}
	zj, err := Command(Zellij, "T-1", []string{"hatch", "run", "T-1"})
	if err != nil || zj[0] != "zellij" || zj[1] != "run" {
		t.Fatalf("zellij command wrong: %v (%v)", zj, err)
	}
	if _, err := Command("nope", "x", []string{"y"}); err == nil {
		t.Fatal("expected error for unknown mux")
	}
}

func TestShJoinQuotes(t *testing.T) {
	got := shJoin([]string{"hatch", "run", "a b", "x'y"})
	if !strings.Contains(got, "'a b'") {
		t.Fatalf("space arg not quoted: %s", got)
	}
}
