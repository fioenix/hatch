package orchestrator

// Per-kind adapters. Flags mirror docs/10-agent-adapters.md. Capability is
// controlled per agent via registry `sandbox`/`approval` hints.

// claudeAdapter drives Claude Code headlessly: `claude -p`.
type claudeAdapter struct{}

func (claudeAdapter) Kind() string { return "claude" }
func (claudeAdapter) Build(req RunRequest) Invocation {
	args := []string{program(req.Agent, "claude"), "-p", req.Prompt, "--output-format", "json"}
	if req.Agent.Model != "" {
		args = append(args, "--model", req.Agent.Model)
	}
	mode := req.Agent.Approval
	if mode == "" {
		mode = "acceptEdits"
	}
	args = append(args, "--permission-mode", mode)
	return Invocation{Args: args, Headless: true}
}

// codexAdapter drives Codex headlessly: `codex exec`.
type codexAdapter struct{}

func (codexAdapter) Kind() string { return "codex" }
func (codexAdapter) Build(req RunRequest) Invocation {
	sandbox := req.Agent.Sandbox
	if sandbox == "" {
		sandbox = "workspace-write"
	}
	args := []string{program(req.Agent, "codex"), "exec", req.Prompt,
		"-s", sandbox, "--skip-git-repo-check", "--json"}
	if req.Agent.Model != "" {
		args = append(args, "-m", req.Agent.Model)
	}
	return Invocation{Args: args, Headless: true}
}

// geminiAdapter drives Gemini CLI headlessly: `gemini -p`.
type geminiAdapter struct{}

func (geminiAdapter) Kind() string { return "gemini" }
func (geminiAdapter) Build(req RunRequest) Invocation {
	mode := req.Agent.Approval
	if mode == "" {
		mode = "auto_edit"
	}
	args := []string{program(req.Agent, "gemini"), "-p", req.Prompt,
		"--approval-mode", mode, "--output-format", "json"}
	if req.Agent.Model != "" {
		args = append(args, "-m", req.Agent.Model)
	}
	return Invocation{Args: args, Headless: true}
}

// kiroAdapter drives Kiro CLI headlessly: `kiro-cli chat --no-interactive`.
// Requires KIRO_API_KEY in the environment (passed through, not set here).
type kiroAdapter struct{}

func (kiroAdapter) Kind() string { return "kiro" }
func (kiroAdapter) Build(req RunRequest) Invocation {
	args := []string{program(req.Agent, "kiro-cli"), "chat", "--no-interactive", req.Prompt}
	return Invocation{Args: args, Headless: true, Note: "requires KIRO_API_KEY in environment"}
}

// mockAdapter drives the hatch-mock test agent: `hatch-mock --prompt …`.
// Used to exercise the real spawn/capture path without a live agent CLI.
type mockAdapter struct{}

func (mockAdapter) Kind() string { return "mock" }
func (mockAdapter) Build(req RunRequest) Invocation {
	return Invocation{
		Args:     []string{program(req.Agent, "hatch-mock"), "--prompt", req.Prompt},
		Headless: true,
	}
}

// manualAdapter represents agents with no headless contract: it produces a
// handoff prompt instead of spawning anything.
type manualAdapter struct {
	kind   string
	reason string
}

func (m manualAdapter) Kind() string { return m.kind }
func (m manualAdapter) Build(req RunRequest) Invocation {
	return Invocation{Headless: false, Note: m.reason, StdinStr: req.Prompt}
}
