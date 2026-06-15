# Security Policy

## Reporting Vulnerabilities

If you discover a security vulnerability in overclaud, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email: tangduyphuong@gmail.com with subject "overclaud security"
3. Include: description, reproduction steps, potential impact

We aim to respond within 48 hours.

## Scope

overclaud is an instruction optimization skill. It:
- Reads and writes local files (CLAUDE.md, settings.json, rules/)
- Never makes external API calls or network requests
- Never collects, stores, or transmits user data
- Includes a PostToolUse hook that only outputs a hardcoded system message

## Hook Security

The auto-suggest hook processes `$TOOL_INPUT` through `jq` (JSON parser) before any string operations. User input never appears in the hook's output. The output is a hardcoded JSON system message.

## Hatch (the orchestrator in `hatch/`)

Hatch is a **local, file-based** tool: state lives in `.hatch/` and the repo;
there is no server and no telemetry. It does, however, execute programs, so the
following are deliberate **trust boundaries** — treat a `.hatch/` workspace and
its `registry.yaml`/`workflow.yaml` as trusted, code-reviewed inputs (like a
`Makefile` or CI config):

- **Gate commands.** `workflow.yaml` gates of `type: command` run via `sh -c`
  (e.g. `make test`). Anyone who can commit `workflow.yaml` can run commands on
  a machine that runs `hatch gate`/`move`. Review workflow changes in PRs.
- **Agent execution.** The orchestrator spawns the agent CLIs named in
  `registry.yaml` (`exec`, no shell — arguments are passed directly, not
  interpolated into a shell string). Only list agents you trust.
- **Credentials.** Agent API keys are read from the environment at spawn time
  (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `KIRO_API_KEY`) and
  are **never** written to the repo, ledger, or config. Never commit secrets;
  `registry.yaml` references env var names only.
- **Transcripts.** Raw agent stdout/stderr is captured to `.hatch/runs/` and the
  ledger summary. If an agent prints a secret, it lands there — output redaction
  is currently out of scope (tracked in `hatch/docs/17-pre-implementation.md`).
  Keep `.hatch/runs/` out of public artifacts if your agents echo sensitive data.
- **Path safety.** Ticket ids, channel names and run targets are sanitized
  before being used as path segments to prevent traversal; `hatch validate`
  rejects unsafe ticket ids.

Report Hatch vulnerabilities the same way as above (subject "hatch security").
