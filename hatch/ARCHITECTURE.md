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
| **Ports** | `internal/port` (`Board`, `Ledger`, `Bus`, `OnCall`) + `gate.Runner`, `orchestrator.Adapter` | Interfaces the use-case layer depends on; adapters satisfy them. |
| **Adapters** | `internal/store` (filesystem board/ledger/KB), `bus` (filesystem messaging), `gate.ShellRunner` (exec), `orchestrator` per-kind agent adapters + `mock`, `mux` (tmux/zellij), `presence`, `oncall`, `mdfront`, `paths` | Implement ports / wrap the outside world. |
| **Driving adapters** | `cmd/hatch`, `internal/cli`, `internal/tui`, `cmd/hatch-mock` | The composition root: parse input, wire concrete adapters into use cases. |

## Ports in practice

The ports live in **`internal/port`** (`Board`, `Ledger`, `Bus`, `OnCall`),
plus consumer-local ports where they belong to one boundary (`gate.Runner`,
`orchestrator.Adapter`). Infrastructure packages provide adapters that satisfy
them, asserted at compile time (`var _ port.Board = (*store.Board)(nil)`).

- **`wf.Engine`** — the workflow engine holds `port.Board/Ledger/Bus/OnCall`
  and exposes `Move`/`Escalate` as methods. It imports **no infrastructure**
  (only `port` + `model` + `config` + `gate`). `engineFor(ws)` in the CLI wires
  the concrete adapters.
- **`orchestrator.Orchestrator`** — holds `port.Ledger` + `port.Bus`; `Run`/
  `Execute` are methods. The agent boundary itself is `orchestrator.Adapter`
  (Claude/Codex/Gemini/Kiro + `mock` test adapter) — textbook ports & adapters.
- **`gate.Runner`** — gate command execution sits behind a port; `ShellRunner`
  is the production adapter, fakes are used in tests (no real `sh`).
- **`metrics.Compute(port.Ledger)`** — reads through the ledger port.

### The command/projection boundary (a deliberate Lean line)

Use cases that **mutate** state or **spawn** agents go through ports
(`wf`, `orchestrator`, `gate`). **Read-only reporting projections** that
aggregate several sources at once (`ceremony`, plus the `report`/`cost`/`budget`
CLI views) are composed at the root and read concrete adapters directly. Forcing
them through ports would require leaking the messaging value types (`bus.Message`)
into a port or relocating domain types — ceremony alone reads ledger **and**
board **and** bus decisions **and** KB. That is abstraction for its own sake; we
draw the line at the command path, where decoupling and testability actually pay
off. If a second backend ever appears, the projections move behind ports then.

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
