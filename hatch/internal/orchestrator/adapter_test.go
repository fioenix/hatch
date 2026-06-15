//go:build hatch_legacy

package orchestrator

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

func build(kind string, mutate func(*model.Agent)) Invocation {
	a := model.Agent{ID: kind, Kind: kind}
	if mutate != nil {
		mutate(&a)
	}
	return AdapterFor(kind).Build(RunRequest{Agent: a, Prompt: "do X"})
}

func TestClaudeInvocation(t *testing.T) {
	inv := build("claude", func(a *model.Agent) { a.Model = "opus"; a.Approval = "plan" })
	got := strings.Join(inv.Args, " ")
	for _, want := range []string{"claude", "-p", "do X", "--output-format json", "--model opus", "--permission-mode plan"} {
		if !strings.Contains(got, want) {
			t.Errorf("claude invocation missing %q in %q", want, got)
		}
	}
	if !inv.Headless {
		t.Error("claude should be headless")
	}
}

func TestCodexDefaultsSandbox(t *testing.T) {
	inv := build("codex", nil)
	got := strings.Join(inv.Args, " ")
	if !strings.Contains(got, "codex exec") || !strings.Contains(got, "-s workspace-write") {
		t.Errorf("codex invocation wrong: %q", got)
	}
}

func TestAgyAndKiro(t *testing.T) {
	a := strings.Join(build("agy", nil).Args, " ")
	if !strings.Contains(a, "agy -p") || strings.Contains(a, "--output-format") {
		t.Errorf("agy invocation wrong: %q", a)
	}
	y := strings.Join(build("agy", func(ag *model.Agent) { ag.Approval = "yolo" }).Args, " ")
	if !strings.Contains(y, "--dangerously-skip-permissions") {
		t.Errorf("agy yolo→--dangerously-skip-permissions missing: %q", y)
	}
	k := build("kiro", nil)
	if !strings.Contains(strings.Join(k.Args, " "), "kiro-cli chat --no-interactive") {
		t.Errorf("kiro invocation wrong: %v", k.Args)
	}
	if !strings.Contains(k.Note, "KIRO_API_KEY") {
		t.Errorf("kiro should note KIRO_API_KEY, got %q", k.Note)
	}
}

func TestManualAndAntigravityNotHeadless(t *testing.T) {
	for _, kind := range []string{"manual", "antigravity", "shell"} {
		inv := AdapterFor(kind).Build(RunRequest{Agent: model.Agent{Kind: kind}, Prompt: "p"})
		if inv.Headless {
			t.Errorf("%s should not be headless", kind)
		}
	}
}

func TestCmdOverride(t *testing.T) {
	inv := build("claude", func(a *model.Agent) { a.Cmd = "claude-canary" })
	if inv.Args[0] != "claude-canary" {
		t.Errorf("expected cmd override, got %q", inv.Args[0])
	}
}
