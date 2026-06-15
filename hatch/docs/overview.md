# Overview — bản đồ tổng Hatch (plaintext)

Legend: **✓** đã implement (code + test) · **◇** mới thiết kế (docs 13–14).

## 1. Bức tranh toàn cảnh

```
┌─ HATCH ─ một bầy coding-agent vận hành như squad người, trên file + git ──────────┐
│                                                                                    │
│  SSOT (.hatch/)                                                                    │
│  charter(L0) · roles(L1) · context(L2) · registry(ai-giữ-vai) · workflow(quy trình)│
│        │  ✓ hatch compile  (1 nguồn → N surface, manifest bắt stale)               │
│        ▼                                                                           │
│  SURFACES  ✓  CLAUDE.md · AGENTS.md · GEMINI.md · .kiro/steering   (L0+L1+con trỏ) │
│        │  nạp instruction native                                                   │
│        ▼                                                                           │
│  AGENTS  ✓  claude · codex · gemini · kiro      (mỗi con 1 process, KHÔNG chung RAM)│
│     ▲   │                                                                          │
│     │   │ claim/move ──►  WORKFLOW ENGINE ✓  (transition · gate · no-self-review · │
│     │   │                 WIP · deps · auto-escalate)                              │
│     │   ▼                                                                          │
│  ORCHESTRATOR ✓  run · plan · watch   ── spawn headless ─┘                         │
│     (catch-up: đọc inbox + recall trước khi vào việc)                              │
│                                                                                    │
│  STORES = DATABASE (filesystem + git):                                             │
│   ✓ board/ (state=thư mục) · ✓ ledger/ (audit) · ✓ kb/ (tri thức) · ✓ bus/ (thoại)│
│                                                                                    │
│  ── lớp "con người" ───────────────────────────────────────────────────────────  │
│  COMMUNICATION ✓  DM · #channel · thread · @mention · search/recall                │
│  HỎI & HỌP     ✓  ask (đồng bộ) · convene (+ tie-breaker) · decision→ADR           │
│  CEREMONIES    ✓  standup · retro · planning · demo · grooming                     │
│  COLLAB        ✓  pairing(driver/navigator) · mob(driver xoay vòng)                │
│  ỨNG CỨU       ✓  escalation · on-call rotation · incident workflow               │
│  CAPACITY      ✓  presence(available/busy/paused/offline) → phân việc theo tải     │
│  OBSERVE       ✓  transcript + hatch logs -f · TUI board(board+live+feed) · chat   │
│                ✓  --mux=tmux|zellij (mỗi run một pane thật)                         │
│  DOCS/KB       ✓  doc templates (new/lint) · Obsidian KB (link/backlinks/graph)    │
│  ── lớp QUẢN TRỊ (Founder/CEO/CTO) ──────────────────────────────────────────────  │
│  ✓ workload · ✓ performance · ✓ budget/lương(track) · ✓ org-chart/DoA              │
│  ✓ external deps · ✓ heartbeat(hatch tick) · ✓ stakeholder report(hatch report)    │
│  ◇ còn lại: auto-pause khi vượt budget (đã chốt track-only), parallel-watch worktree│
└────────────────────────────────────────────────────────────────────────────────┘

3 kho tri thức:  SSOT = config VÀO  ·  KB = tri thức VÀO+RA  ·  ledger = sự kiện RA
                 (bus = đối thoại; board = trạng thái)
```

## 2. Bản đồ lệnh CLI (theo nhóm)

```
SETUP/SSOT     init · compile [--check] · validate · sync · hook install
BOARD/WORK     ticket new|claim|move|show|extdep · status · standup · gate
ORCH/OBSERVE   run [--claim --worktree --mux=tmux|zellij] · plan · watch [--parallel N]
               tick · logs [-f] · board (TUI)
KNOWLEDGE      kb add|query|index|link|backlinks|graph|open · doc types|new|lint
COMMUNICATION  msg · inbox · thread · channel ls|show|join|leave|members · search · chat(TUI)
DIALOGUE       ask · convene [--decider] · escalate
COLLAB         pair · mob
CEREMONY       ceremony standup|retro|planning|demo|grooming
CAPACITY/OPS   presence [set] · oncall [set|rotate]
MANAGE         workload · perf · cost · budget · report · org
◇ còn lại      auto-pause budget (chốt track-only) · parallel-watch worktree · daemon
```

## 3. "Một ngày của squad" (luồng dệt mọi mảnh)

```
  Founder ─ hatch plan ─► Conductor bẻ epic → tickets (role/priority/deps)   [ledger]
                                   │
  ceremony grooming ─► chuốt backlog (thiếu role/acceptance?)
                                   │
  watch/run ─► chọn agent RẢNH & ít WIP (presence+capacity) ─► claim (git push=lock)
       │  catch-up: đọc inbox + recall #channel liên quan
       ▼
  Implementer code ──┬─ ask reviewer "chốt X?" (đồng bộ qua bus)
                     ├─ pair/mob nếu cần nhiều đầu
                     └─ convene "thiết kế Y" → DECISION → ADR vào kb/
       │ move → review  [gate: test·lint·handoff]   (fail 2 lần → auto escalate → on-call)
       ▼
  Reviewer (≠ implementer) ─ gate DoD·no-self-review·human-merge ─► done   [ledger]
                                   │
  ceremony standup (digest theo agent + blockers → #standup)
  ceremony demo (showcase việc done → #demo)
  ceremony retro (done/blocks/gate-fail/decisions + đề bạt KB→SSOT)
       │
  ◇ management: workload/perf/budget mỗi chu kỳ → report cho stakeholder
```

## 4. Đối ứng squad người (cô đọng)

```
  onboarding handbook   = compiler → CLAUDE.md/AGENTS.md theo vai
  Jira board / sprint   = board/ + workflow (8 template PDLC)
  Slack                 = bus (DM·channel·thread·@mention·search)
  hỏi đồng nghiệp / họp  = ask · convene
  pair / mob            = pair · mob
  standup/retro/demo    = ceremony
  on-call / sự cố       = oncall · escalate · incident workflow
  wiki / ADR            = kb/
  nhật ký / git log     = ledger/
  ai rảnh, PTO          = presence
  ◇ EM/CEO dashboard    = workload · perf · budget/lương · org-chart/DoA
```

> Chi tiết: kiến trúc [01](01-architecture.md) · workflow [05](05-workflow.md) · KB [09](09-knowledge-base.md) · adapters [10](10-agent-adapters.md) · communication [11](11-communication.md) · ceremonies/escalation [12](12-ceremonies-escalation.md) · management [13](13-management.md) · org/cadence [14](14-org-and-cadence.md).
