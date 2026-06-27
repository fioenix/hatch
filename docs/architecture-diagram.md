# Sơ đồ kiến trúc & workflow

> **Mô hình embedded-harness** (xem [doc 20](20-embedded-harness-pivot.md)). Coding agent là entrypoint và **tự lái**; Hatch là lớp nền chung (chat = comms + backlog, KB, ledger) mà agent với tới **qua MCP**. Hatch **không** spawn/điều khiển agent.

Bản plaintext là chính (đọc thẳng trong terminal/diff). Khối Mermaid bên dưới cho bản render trên GitHub.

## 1. Kiến trúc hệ thống (plaintext)

```
                       .hatch/  —  SINGLE SOURCE OF TRUTH
      ┌────────────────────────────────────────────────────────────────┐
      │ charter.md(L0)   roles/*.md(L1)   context/(L2)                   │
      │ registry.yaml (ai giữ vai gì)     workflow.yaml (quy trình)      │
      └─────────────────────────────┬──────────────────────────────────-┘
                                    │  hatch compile
                                    ▼
      ┌──────────────────────────────────────┐   ┌──────────────────────────┐
      │ SURFACES (per-agent, sinh ra)         │   │ MCP REGISTRATION (sinh ra)│
      │ CLAUDE.md · AGENTS.md · GEMINI.md     │   │ .mcp.json (Claude)        │
      │ .kiro/steering/                       │   │ .kiro/settings/mcp.json   │
      │ = protocol PROSE: workflow + chat     │   │ .hatch/mcp/*  (codex, agy)│
      │   etiquette + DoD self-check          │   └────────────┬──────────────┘
      │   (+ khối Orchestrator cho lead)      │                │ "chạy: hatch mcp --as <id>"
      └───────────────────┬──────────────────┘                │
                          │ agent đọc khi khởi động           │
                          ▼                                    ▼
      ┌──────────────────────────────────────┐      ┌────────────────────────┐
      │ CODING AGENTS  (ENTRYPOINT, tự lái;   │      │   HATCH MCP SERVER      │
      │ mỗi con 1 process, KHÔNG chung RAM)   │─────►│  hatch mcp --as <id>    │
      │ Claude Code · Codex · agy · Kiro      │ MCP  │  (stdio, 1 instance/    │
      │  • lead = Conductor (mở thread/task)  │ tools│   agent, đúng danh tính)│
      └──────────────────────────────────────┘      └───────────┬─────────────┘
                                                                 │ đọc/ghi
                ┌────────────────────────────────────────────────┘
                ▼
      ┌─────────────────────────────────────────────────────────────────┐
      │                 HỆ THỐNG FILE = DATABASE  (git)                   │
      │  bus/   = CHAT  →  comms + BACKLOG  (1 thread = 1 task)           │
      │           channel · thread · @mention · search · inbox           │
      │  kb/    = Knowledge Base  (đọc & GHI chung: decision/learning)    │
      │  ledger/= append-only audit (who/what/why)                       │
      └─────────────────────────────────────────────────────────────────┘
                ▲ read-only
                │
      ┌─────────────────────────────────────┐
      │ OBSERVE (con người xem, không lái)   │   hatch msg  ──► chèn ý kiến vào chat
      │ hatch board · hatch chat · hatch status   (read-only views)      │
      └─────────────────────────────────────┘

  Ba kho tri thức:  SSOT = config VÀO  ·  KB = tri thức VÀO+RA  ·  ledger = sự kiện RA
  (Archived sau -tags hatch_legacy: orchestrator spawn · workflow-engine · ceremonies…)
```

## 2. Vòng đời một task = một thread chat (quy ước, KHÔNG phải engine)

```
   chat_open               chat_post (tiến độ)        @tag reviewer        chat_post "done"
   ┌────────┐   bắt tay    ┌─────────────┐  xong việc  ┌────────┐  duyệt   ┌──────┐
   │  OPEN  │ ───────────► │ IN-PROGRESS │ ──────────► │ REVIEW │ ───────► │ DONE │
   └────────┘  (mở task)   └─────────────┘             └────────┘          └──────┘
                                 │  ▲                       │
                       post      │  │ gỡ kẹt                │ changes-requested
                      "block"    ▼  │ (post tiếp)           │ (@tag lại tác giả)
                            ┌─────────┐ ◄───────────────────┘
                            │ BLOCKED │
                            └─────────┘
   • Trạng thái SUY RA từ hội thoại (post type done/block/decision) — không có lane-engine.
   • workflow.yaml chỉ là PROSE hướng dẫn (đã compile vào CLAUDE.md…); ai làm vai gì theo đó.
   • Gate = Definition-of-Done agent TỰ chạy & xác nhận (make test/lint, no-self-review, human-merge).
```

## 3. Một vòng trao đổi giữa agent (qua MCP + bus)

```
  Conductor (Claude)              Hatch bus (#export-csv)            Codex
        │  chat_open "Export CSV   │                                  │
        │   @codex stream giúp" ──► │  thread tạo, @codex vào "To"     │
        │                          │ ◄── chat_inbox ─────────────────-┤ (đầu session)
        │                          │     thấy @mention                 │
        │                          │ ◄── chat_read #export-csv ───────-┤ hiểu nhiệm vụ
        │                          │            code + test            │
        │                          │ ◄── chat_post "PR #42, @claude    │
        │ ◄── chat_inbox ──────────┤     review" (reply_to=root) ──────┘
        │  đọc, REVIEW (≠ tác giả) │
        │  kb_add "ADR: streaming" │  (tri thức đáng giữ → KB)
        │  chat_post "done" ──────► │  thread khép (trạng thái: done)
        ▼                          ▼
  (không ai spawn ai — mọi trao đổi bất đồng bộ, agent trả lời khi đang chạy)
```

---

## Bản Mermaid (render trên GitHub)

```mermaid
flowchart TB
  classDef ssot fill:#1f2937,color:#fff,stroke:#111;
  classDef surf fill:#2563eb,color:#fff,stroke:#1d4ed8;
  classDef store fill:#eef2ff,stroke:#4338ca,color:#3730a3;
  classDef eng fill:#fff,stroke:#2563eb,color:#1e3a8a,stroke-width:2px;
  classDef obs fill:#f0fdf4,stroke:#16a34a,color:#166534;

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
  WF --> COMPILER
  COMPILER --> SURF["Surfaces per-agent<br/>CLAUDE.md · AGENTS.md · GEMINI.md · .kiro<br/>(protocol prose + DoD + orchestrator)"]:::surf
  COMPILER --> MCPREG["MCP registration<br/>.mcp.json · .kiro/settings/mcp.json · .hatch/mcp/*"]:::surf

  subgraph AGENTS["Coding agents — ENTRYPOINT, tự lái (1 process/con)"]
    direction LR
    A1["Claude Code<br/>(lead = Conductor)"]
    A2["Codex"]
    A3["agy"]
    A4["Kiro"]
  end
  SURF --> AGENTS
  MCPREG -. "hatch mcp --as id" .-> MCP

  MCP{{"Hatch MCP server<br/>(stdio, per agent)"}}:::eng
  AGENTS == "MCP tools" ==> MCP

  subgraph STORE["Hệ thống file = database (git)"]
    direction LR
    BUS[("bus/ — CHAT = comms + backlog<br/>thread = task")]:::store
    KB[("kb/ — Knowledge Base")]:::store
    LEDGER[("ledger/ — audit")]:::store
  end
  MCP --> BUS
  MCP --> KB
  MCP --> LEDGER

  subgraph OBS["Observe (read-only, con người)"]
    direction LR
    O1["hatch board"]:::obs
    O2["hatch chat"]:::obs
    O3["hatch status"]:::obs
  end
  BUS --> OBS
  LEDGER --> OBS
```

```mermaid
stateDiagram-v2
  direction LR
  [*] --> open: chat_open (mở task)
  open --> in_progress: bắt tay (chat_post)
  in_progress --> review: xong + @tag reviewer
  review --> done: reviewer duyệt + post "done"
  review --> in_progress: changes-requested
  in_progress --> blocked: post "block"
  blocked --> in_progress: gỡ kẹt
  done --> [*]
  note right of review
    Trạng thái suy ra từ hội thoại.
    Gate = DoD agent tự kiểm.
  end note
```
