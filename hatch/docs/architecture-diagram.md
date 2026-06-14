# Sơ đồ kiến trúc & workflow

Ba góc nhìn của Hatch. Khối Mermaid render trực tiếp trên GitHub; ảnh PNG tương ứng nằm trong `assets/`.

## 1. Kiến trúc hệ thống

SSOT compile xuống từng surface agent; agents điều phối qua board/ledger; KB là bộ nhớ chung đọc-ghi; orchestrator spawn agent headless.

![Kiến trúc hệ thống](assets/arch.png)

```mermaid
flowchart TB
  classDef ssot fill:#2a2b86,color:#fff,stroke:#2a2b86;
  classDef surf fill:#fcaf16,color:#1a1a1a,stroke:#c98a00;
  classDef store fill:#eef,stroke:#2a2b86,color:#2a2b86;
  classDef eng fill:#fff,stroke:#2a2b86,color:#2a2b86,stroke-width:2px;

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
  COMPILER -. "manifest hash → stale detect" .-> MAN[("compiled/.manifest.json")]:::store

  COMPILER --> S1["CLAUDE.md"]:::surf
  COMPILER --> S2["AGENTS.md<br/>(Codex/Antigravity)"]:::surf
  COMPILER --> S3["GEMINI.md"]:::surf
  COMPILER --> S4[".kiro/steering/"]:::surf

  subgraph AGENTS["Agents — mỗi con 1 process, KHÔNG chung RAM"]
    direction LR
    A1["Claude Code"]
    A2["Codex"]
    A3["Gemini"]
    A4["Kiro"]
  end
  S1 --> A1
  S2 --> A2
  S3 --> A3
  S4 --> A4

  ORCH{{"orchestrator<br/>run · plan · watch"}}:::eng
  ORCH == "spawn headless<br/>(claude -p · codex exec · …)" ==> AGENTS

  ENGINE{{"workflow engine + gates<br/>(transition · no-self-review · WIP · deps)"}}:::eng
  WF --> ENGINE

  subgraph STORE["Hệ thống file = database"]
    direction LR
    BOARD[("board/ — tickets<br/>vị trí thư mục = trạng thái")]:::store
    LEDGER[("ledger/ — append-only audit")]:::store
    KB[("kb/ — Knowledge Base<br/>đọc & ghi chung")]:::store
  end

  AGENTS == "claim / move" ==> ENGINE
  ENGINE --> BOARD
  ENGINE --> LEDGER
  AGENTS -- "tra cứu + đóng góp" --> KB
  KB -. "promote khi chín (retro)" .-> CX
```

## 2. Workflow — máy trạng thái ticket (template `scrum`)

Lane = thư mục trong `board/`. Mỗi mũi tên là một transition do `workflow.yaml` định nghĩa; nhãn ghi rõ ai được làm + gate phải qua.

![Workflow ticket](assets/workflow.png)

```mermaid
stateDiagram-v2
  direction LR
  [*] --> backlog
  backlog --> in_progress: claim · implementer/tester (deps done)
  in_progress --> review: handoff — gates tests/lint/handoff-note
  review --> done: done · reviewer — gates dod/no-self-review/human-merge
  review --> in_progress: changes-requested
  in_progress --> blocked: block
  blocked --> in_progress: unblock
  done --> [*]

  note right of backlog: vị trí thư mục = trạng thái
  note right of review: no-self-review — reviewer khác implementer
```

## 3. Vòng đời một ticket (sequence)

Conductor lập kế hoạch → Implementer claim & làm → gate → Reviewer duyệt. Mọi bước để lại ledger; tri thức vào KB.

![Vòng đời ticket](assets/sequence.png)

```mermaid
sequenceDiagram
  autonumber
  actor H as Human
  participant CC as Claude (Conductor)
  participant B as board/
  participant CX as Codex (Implementer)
  participant K as kb/
  participant RV as Claude (Reviewer)
  participant L as ledger/

  H->>CC: hatch plan
  CC->>B: ticket new T-001 (role=implementer)
  CC->>L: note · why=kế hoạch sprint

  Note over CX: hatch run --claim T-001
  CX->>B: claim → in-progress (git push = lock)
  CX->>L: claim
  CX->>K: đọc ADR/learnings liên quan (L2)
  CX->>CX: code + test trong scope
  CX->>B: move → review  ✓gates
  CX->>L: handoff (đã làm/còn/cần)

  Note over RV: no-self-review: RV ≠ CX
  RV->>B: move → done  ⊙human-merge
  RV->>L: review · approved
  RV->>K: ghi learning mới
```
