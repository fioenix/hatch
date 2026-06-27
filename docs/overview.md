# Overview — bản đồ tổng Hatch (plaintext)

> **Mô hình hiện hành = embedded harness** (xem [doc 20](20-embedded-harness-pivot.md)). Coding agent là entrypoint và **tự lái mình**; Hatch là lớp nền chung (chat = comms + backlog · KB · ledger) mà agent với tới qua **MCP server**. Hatch **không** spawn, **không** điều khiển agent. Các doc 00–19 mô tả thiết kế GỐC (trước pivot); khi mâu thuẫn, doc 20 thắng.

Legend: **✓** đã implement (code + test) · **⌖** archived sau build tag `hatch_legacy` (không vào binary mặc định, khôi phục được).

## 1. Bức tranh toàn cảnh

```
┌─ HATCH ─ embedded harness cho một bầy coding-agent, trên file + git ───────────────┐
│                                                                                    │
│  SSOT (.hatch/)                                                                    │
│  charter(L0) · roles(L1) · context(L2) · registry(ai-giữ-vai) · workflow(quy trình)│
│        │  ✓ hatch compile  (1 nguồn → N surface, manifest bắt stale)               │
│        ▼                                                                           │
│  SURFACES  ✓  CLAUDE.md · AGENTS.md · GEMINI.md · .kiro/steering                   │
│        │  PROTOCOL prose: workflow + chat etiquette + DoD self-check               │
│        │  + khối "orchestrator" cho lead  +  ĐĂNG KÝ MCP per-agent                  │
│        │     (.mcp.json · .kiro/settings/mcp.json merge; snippet .hatch/mcp/* codex/agy)
│        ▼                                                                           │
│  AGENTS  ✓  claude · codex · agy · kiro   (entrypoint; TỰ LÁI; mỗi con 1 process)  │
│        │                                                                           │
│        │  với tới chat + KB qua  ── hatch mcp --as <agent> (stdio) ──┐             │
│        ▼                                                              ▼             │
│  MCP TOOLS ✓  whoami · chat_open · chat_post · chat_read · chat_inbox ·             │
│               chat_search · chat_channels · kb_add · kb_search                     │
│                                                                                    │
│  STORES = DATABASE (filesystem + git):                                             │
│   ✓ bus/ (chat = comms + backlog: 1 thread = 1 task) · ✓ kb/ (tri thức) ·          │
│   ✓ ledger/ (audit append-only)                                                    │
│                                                                                    │
│  ── lớp "quan sát của người" (READ-ONLY) ──────────────────────────────────────── │
│  ✓ hatch board  TUI: THREADS + CHAT + ledger ACTIVITY                              │
│  ✓ hatch chat   TUI: viewer hội thoại kiểu Slack                                   │
│  ✓ hatch status tóm tắt thread (task) + roster agent                               │
│  ✓ hatch msg    người chèn ý kiến vào chat                                         │
│                                                                                    │
│  ── operator tự-lái = ARCHIVED sau `hatch_legacy` ──────────────────────────────── │
│  ⌖ run · plan · watch · tick · orchestrator · workflow-engine(gate/escalate/ticket)│
│  ⌖ ceremony/standup · ask/convene · pair/mob · presence · oncall · cost/budget ·   │
│  ⌖ workload/perf · report   (không vào binary mặc định; go build -tags hatch_legacy)│
└────────────────────────────────────────────────────────────────────────────────┘

3 kho tri thức:  SSOT = config VÀO  ·  KB = tri thức VÀO+RA  ·  ledger = sự kiện RA
                 (bus = chat: đối thoại VÀ backlog — mỗi thread = một task)
```

## 2. Bản đồ lệnh CLI (theo nhóm)

```
SETUP/SSOT     init · compile [--check] · validate · sync · hook
                 compile sinh: protocol prose (workflow + chat etiquette + DoD self-check
                 + khối orchestrator cho lead) + đăng ký MCP per-agent
HARNESS        mcp --as <agent>        (stdio MCP server: chat + KB tools cho agent)
OBSERVE (RO)   status · board (TUI) · chat (TUI) · logs
CHAT/BACKLOG   msg · inbox · thread · channel ls|show|join|leave|members · search
KNOWLEDGE      kb add|query|index|link|backlinks|graph|open · doc
MISC           org · doctor
⌖ archived     run · plan · watch · tick · ceremony/standup · ask/convene · pair/mob ·
   (hatch_legacy)  presence · oncall · cost/budget · workload/perf · report · gate/escalate/ticket
```

## 3. "Một ngày của squad" (luồng dệt mọi mảnh)

```
  hatch init  →  hatch compile  (SSOT → surface + đăng ký MCP)
       │
  Người mở MỘT coding agent NGAY TRONG workspace (vd Claude Code).
  CLAUDE.md (protocol + khối orchestrator) + .mcp.json đã wire nó vào chat + KB chung.
       │  agent đầu tiên = Conductor/orchestrator + có thể kiêm vai khác
       ▼
  Conductor bẻ việc → MỞ MỘT THREAD cho mỗi task qua MCP (chat_open) → brief → @tag agent phù hợp
       │
  Agent được tag (đang chạy) đọc thread → làm → brief kết quả TRONG CÙNG THREAD (chat_post/reply)
       │  recall bằng chat_search / kb_search; ghi quyết định/bài học bằng kb_add
       │  trạng thái task suy ra từ hội thoại (post type done/block/decision) — không lane/gate engine
       ▼
  Reviewer (≠ implementer) tự chạy DoD self-check → báo done trong thread
       │
  Tất cả ASYNC kiểu Slack: agent chỉ phản hồi khi nó đang chạy; không ai spawn ai; Hatch không spawn.
       │
  Người xem bằng hatch chat (board/watch = alias) / hatch status; chèn ý kiến bằng hatch msg.
```

## 4. Đối ứng squad người (cô đọng)

```
  onboarding handbook + working agreements = compiler → CLAUDE.md/AGENTS.md (protocol prose) theo vai
  Slack của đội                            = bus (chat: channel · thread · @mention · search · inbox)
                                              — đồng thời là backlog: mỗi thread = một task
  cánh cửa vào Slack + wiki chung          = MCP server (hatch mcp --as <agent>)
  wiki / ADR / bộ não chung                = kb/  (đọc VÀ ghi qua MCP)
  nhật ký / git log                        = ledger/
  Engineering Manager / Scrum Master       = agent lead/Conductor (LÀ một agent tự lái, không phải Hatch)
  bảng quan sát của quản lý                = hatch chat (live, board/watch=alias) / status (read-only)
  ⌖ EM/CEO dashboard (workload/perf/budget)= archived sau hatch_legacy
```

> Chi tiết: **mô hình hiện hành [20](20-embedded-harness-pivot.md)** · kiến trúc [01](01-architecture.md) · context-compiler [04](04-context-compiler.md) · KB [09](09-knowledge-base.md) · adapters [10](10-agent-adapters.md). Các doc 02–19 cho bối cảnh/ý tưởng nền (thiết kế gốc trước pivot).
