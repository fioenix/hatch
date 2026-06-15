# Hatch — Claude Code plugin

Bundles the Hatch **embedded harness** for Claude Code:

- **MCP server** (`.mcp.json`) — launches `hatch mcp`, exposing the shared squad
  chat (= communication + backlog) and knowledge base as tools.
- **Skill** (`skills/hatch-chat`) — teaches the chat etiquette (one task = one
  thread, `@mention`, recall before re-deriving, DoD self-check).
- **Slash command** (`/hatch`) — sync with the squad: read your inbox and open
  task threads, then summarize what needs you.

## Prerequisites

The `hatch` binary must be on your `PATH`, and you must run Claude Code from
inside a Hatch workspace (a directory with `.hatch/`, created by `hatch init`).

```sh
go install github.com/fioenix/overclaud/hatch/cmd/hatch@latest   # or: make build
hatch init                                                       # in your repo
```

The MCP server posts under the workspace's Claude-kind agent automatically (or
set `HATCH_AGENT`, or pass `--as`). No per-project config is baked into the
plugin.

## Install

During development, point Claude Code at this directory:

```sh
claude --plugin-dir /path/to/overclaud/hatch/plugin
```

Or add the marketplace and install by name:

```
/plugin marketplace add fioenix/overclaud
/plugin install hatch@hatch
```

## Use

1. Open Claude Code in your Hatch workspace.
2. Run `/hatch` to sync with the squad.
3. Collaborate through the `hatch` chat tools — a thread is a task; `@mention`
   teammates; `kb_add`/`kb_search` for shared memory.
