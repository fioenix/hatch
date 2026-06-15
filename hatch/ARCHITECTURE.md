# Architecture

Hatch follows a **Lean Hexagonal** (ports & adapters) design: a pure domain at
the center, use-case services that depend on **ports** (interfaces), and
infrastructure as **adapters** behind those ports. "Lean" = we only introduce a
port where it buys decoupling or testability; we don't abstract things we will
never swap (the filesystem *is* the database, by design).

## The dependency rule

Dependencies point **inward**. Inner layers never import outer ones.

```
   driving adapters            cmd/hatch, internal/cli, internal/tui, cmd/hatch-mock
        │ wire + call
        ▼
   use cases (application)     wf · compile · metrics · ceremony · orchestrator · docs · config
        │ depend on
        ▼
   ports (interfaces)          wf.Board · wf.Ledger · gate.Runner · orchestrator.Adapter
        ▲ implemented by
        │
   adapters (infrastructure)   store · bus · gate.ShellRunner · orchestrator(claude/codex/…/mock)
                               presence · oncall · mux · mdfront · paths
        │ all built on
        ▼
   domain (pure)               model   (entities + value objects, zero IO imports)
```

## Layers

| Layer | Packages | Rule |
|---|---|---|
| **Domain** | `internal/model` | Pure data + invariants. Imports nothing from the project. |
| **Use cases** | `internal/wf`, `compile`, `metrics`, `ceremony`, `orchestrator`, `docs`, `config` | Orchestrate the domain via **ports**; no direct knowledge of *how* IO happens. |
| **Ports** | `wf.Board`, `wf.Ledger`, `gate.Runner`, `orchestrator.Adapter` | Interfaces declared at the consumer (Go idiom). |
| **Adapters** | `internal/store` (filesystem board/ledger/KB), `bus` (filesystem messaging), `gate.ShellRunner` (exec), `orchestrator` per-kind agent adapters + `mock`, `mux` (tmux/zellij), `presence`, `oncall`, `mdfront`, `paths` | Implement ports / wrap the outside world. |
| **Driving adapters** | `cmd/hatch`, `internal/cli`, `internal/tui`, `cmd/hatch-mock` | The composition root: parse input, wire concrete adapters into use cases. |

## Ports in practice

- **`wf.Board` / `wf.Ledger`** — the workflow engine (the core use case) operates
  on these interfaces, not on `*store.Board`/`*store.Ledger`. `internal/store`
  satisfies them structurally. The engine package imports **no infrastructure**
  (`internal/wf` does not import `internal/store`).
- **`gate.Runner`** — gate command execution sits behind a port; `ShellRunner`
  is the production adapter, fakes are used in tests (no real `sh`).
- **`orchestrator.Adapter`** — each agent CLI (Claude/Codex/Gemini/Kiro) is an
  adapter that builds a headless invocation; `mock` is the test/demo adapter.
  This is textbook ports & adapters at the agent boundary.

## Where we stay Lean (intentional non-ports)

We do **not** hide these behind interfaces, because there is one real
implementation and swapping it is a non-goal:

- The filesystem layout (`paths`) and markdown encoding (`mdfront`) — the
  storage format is the product.
- `bus`, `presence`, `oncall` — small filesystem adapters used directly by
  use-case services acting as coordinators.

If a second backend ever appears (e.g. a server-backed store), the seams above
(`store` behind `wf.Board`/`wf.Ledger`) are where a new adapter slots in without
touching the engine.

## Testing reflects the architecture

- Domain + use cases are unit-tested with fakes through ports (`gate` with a
  fake `Runner`; `wf` with the real `store` satisfying the interfaces).
- `orchestrator` is exercised end-to-end with the **mock** agent adapter.
- `internal/cli` has a full lifecycle integration test driving the real command
  tree (init → compile → ticket → run → logs → report).
