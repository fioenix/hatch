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

func TestGeminiAndKiro(t *testing.T) {
	g := strings.Join(build("gemini", nil).Args, " ")
	if !strings.Contains(g, "gemini -p") || !strings.Contains(g, "--approval-mode auto_edit") {
		t.Errorf("gemini invocation wrong: %q", g)
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
