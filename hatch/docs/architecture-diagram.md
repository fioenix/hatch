# Sơ đồ kiến trúc & workflow

Bản plaintext là chính (đọc thẳng trong terminal/diff). Khối Mermaid bên dưới cho bản render trên GitHub.

## 1. Kiến trúc hệ thống (plaintext)

```
                         .hatch/  —  SINGLE SOURCE OF TRUTH
        ┌──────────────────────────────────────────────────────────┐
        │ charter.md(L0)  roles/*.md(L1)  context/(L2)               │
        │ registry.yaml(ai giữ vai gì)    workflow.yaml(quy trình)   │
        └───────────────┬──────────────────────────┬─────────────────┘
                        │ hatch compile             │ workflow.yaml
                        ▼                            ▼
        ┌──────────────────────────────┐   ┌────────────────────────────┐
        │ SURFACES (per-agent, sinh ra) │   │  WORKFLOW ENGINE + GATES    │
        │ CLAUDE.md  AGENTS.md          │   │  transition · WIP · deps    │
        │ GEMINI.md  .kiro/steering/    │   │  no-self-review · gates     │
        │  (+ compiled/.manifest.json)  │   └──────────────┬──────────────┘
        └───────────────┬───────────────┘                  │ authorise
                        │ nạp L0+L1+con trỏ L2              │
                        ▼                                   │
        ┌──────────────────────────────┐   claim/move      │
        │ AGENTS (mỗi con 1 process,    │◀──────────────────┘
        │ KHÔNG chung RAM)              │
        │ Claude · Codex · Gemini · Kiro│
        └───┬───────────────┬───────────┘
   spawn ▲  │ claim/move    │ tra cứu + đóng góp
headless │  ▼               ▼
 ┌───────┴──────┐   ┌───────────────────────────────────────────────┐
 │ ORCHESTRATOR │   │           HỆ THỐNG FILE = DATABASE             │
 │ run·plan·    │   │  board/  (vị trí thư mục = trạng thái ticket)  │
 │ watch        │   │  ledger/ (append-only audit: who/what/why)     │
 └──────────────┘   │  kb/     (Knowledge Base — đọc & GHI chung)     │
                    └───────────────────────────────────────────────┘
                          kb/ ──promote khi chín (retro)──► context/ (SSOT)

  Ba kho tri thức:  SSOT = config VÀO  ·  KB = tri thức VÀO+RA  ·  ledger = sự kiện RA
```

## 2. Workflow — vòng đời ticket (plaintext, template `scrum`)

```
   ┌─────────┐  claim   ┌─────────────┐  handoff   ┌────────┐  done   ┌──────┐
   │ backlog │ ───────► │ in-progress │ ─────────► │ review │ ──────► │ done │
   └─────────┘ impl/test└─────────────┘ gates:     └────────┘ reviewer└──────┘
                  ▲           │ │        tests·lint·    │      gates: dod·
        unblock   │     block │ │        handoff-note   │      no-self-review·
                  │           ▼ │                       │      human-merge
              ┌─────────┐      │ └───────────────────────┘
              │ blocked │◀─────┘   changes-requested (review → in-progress)
              └─────────┘
   (lane = thư mục trong board/ · mỗi mũi tên = transition trong workflow.yaml)
```

## 3. Vòng đời một ticket qua các agent (plaintext)

```
  Human ── hatch plan ─► Conductor(Claude) ── ticket new T-001 ─► board/ + ledger(note)
                                                                       │
  Implementer(Codex) ── hatch run --claim T-001 ─► claim (git push = lock) ─► ledger(claim)
        │  đọc kb/ (ADR/learnings, L2)  ──►  code + test trong scope
        └─ move → review  [gates: tests·lint·handoff] ─► ledger(handoff: đã làm/còn/cần)
                                                                       │
  Reviewer(Claude ≠ Codex) ── move → done [no-self-review · human-merge] ─► ledger(approved)
                                          └─ ghi learning mới ─► kb/
```

---

## Bản Mermaid (render trên GitHub)

```mermaid
flowchart TB
  classDef ssot fill:#1f2937,color:#fff,stroke:#111;
  classDef surf fill:#2563eb,color:#fff,stroke:#1d4ed8;
  classDef store fill:#eef2ff,stroke:#4338ca,color:#3730a3;
  classDef eng fill:#fff,stroke:#2563eb,color:#1e3a8a,stroke-width:2px;

  subgraph SSOT["SSOT — nguồn canonical (.hatch/)"]
    direction LR
    CH["charter.md (L0)"]:::ssot
    RO["roles/*.md (L1)"]:::ssot
    CX["context/ (L2)"]:::ssot
    REG["registry.yaml"]:::ssot
    WF["workflow.yaml"]:::ssot
  end
  COMPILER{{"hatch compile"}}:::eng
  CH --> COMPILER
  RO --> COMPILER
  CX --> COMPILER
  REG --> COMPILER
  COMPILER -. "manifest → stale detect" .-> MAN[("compiled/.manifest.json")]:::store
  COMPILER --> S1["CLAUDE.md"]:::surf
  COMPILER --> S2["AGENTS.md"]:::surf
  COMPILER --> S3["GEMINI.md"]:::surf
  COMPILER --> S4[".kiro/steering/"]:::surf
  subgraph AGENTS["Agents — mỗi con 1 process"]
    direction LR
    A1["Claude"]
    A2["Codex"]
    A3["Gemini"]
    A4["Kiro"]
  end
  S1 --> A1
  S2 --> A2
  S3 --> A3
  S4 --> A4
  ORCH{{"orchestrator<br/>run · plan · watch"}}:::eng
  ORCH == "spawn headless" ==> AGENTS
  ENGINE{{"workflow engine + gates"}}:::eng
  WF --> ENGINE
  subgraph STORE["Hệ thống file = database"]
    direction LR
    BOARD[("board/")]:::store
    LEDGER[("ledger/")]:::store
    KB[("kb/ — Knowledge Base")]:::store
  end
  AGENTS == "claim / move" ==> ENGINE
  ENGINE --> BOARD
  ENGINE --> LEDGER
  AGENTS -- "tra cứu + đóng góp" --> KB
  KB -. "promote (retro)" .-> CX
```

```mermaid
stateDiagram-v2
  direction LR
  [*] --> backlog
  backlog --> in_progress: claim · impl/test (deps done)
  in_progress --> review: handoff — tests/lint/handoff-note
  review --> done: done · reviewer — dod/no-self-review/human-merge
  review --> in_progress: changes-requested
  in_progress --> blocked: block
  blocked --> in_progress: unblock
  done --> [*]
```
