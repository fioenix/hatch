# 01 — Architecture

## Sáu trụ

```
                    ┌─────────────────────────────┐
                    │      ORCHESTRATOR (act)     │  ← Phase 3: EM/Scrum Master
                    │  plan · run · gate · status │
                    └──────────────┬──────────────┘
                                   │ đọc/ghi
        ┌──────────────────────────┼──────────────────────────┐
        ▼                          ▼                           ▼
 ┌────────────┐           ┌────────────────┐          ┌────────────────┐
 │  CONTEXT   │  compile  │     BOARD      │  ghi vào │     LEDGER     │
 │  (SSOT)    │ ────────► │   (tickets)    │ ───────► │ (audit append) │
 └─────┬──────┘           └───────┬────────┘          └────────────────┘
       │                          │ claim/execute
       ▼ sinh ra                  ▼
 ┌──────────────────────┐   ┌──────────────────────────────────┐
 │ CLAUDE.md / AGENTS.md │   │  AGENTS theo ROLES (registry)    │
 │ .kiro/steering/ …     │◄──│  Architect·Implementer·Reviewer  │
 └──────────────────────┘   └──────────────────────────────────┘
        ▲                                  ▲
        └──────── PROTOCOL (working agreements) ───────┘
```

1. **Context (SSOT)** — nguồn canonical duy nhất.
2. **Compiler** — sinh file instruction native cho từng agent từ SSOT.
3. **Roles + Registry** — định nghĩa vai và gán agent vào vai.
4. **Board** — hàng đợi việc dạng ticket file.
5. **Ledger** — sổ audit append-only.
6. **Protocol** — quy ước phối hợp (claim/lock, handoff, branching, DoD).
7. **Orchestrator** — (Phase 3) CLI điều khiển toàn bộ.

## Layout `.hatch/` trong repo đích

```
.hatch/
├── charter.md              # L0 — mission chung, nhỏ gọn (mọi agent đều nạp)
├── registry.yaml           # roster agent + năng lực + binding vai
│
├── roles/                  # L1 — context theo vai, mỗi vai 1 file
│   ├── architect.md
│   ├── implementer.md
│   ├── reviewer.md
│   ├── tester.md
│   └── ...
│
├── context/                # SSOT — tri thức canonical, compile ra ngoài
│   ├── product/            #   PRD, domain, business rules
│   ├── tech/               #   stack, conventions, kiến trúc
│   └── shared.md           #   những gì mọi vai cần
│
├── board/                  # tickets — mỗi ticket 1 file .md có frontmatter
│   ├── backlog/
│   ├── in-progress/
│   ├── review/
│   └── done/
│
├── ledger/                 # audit append-only (1 file / ngày hoặc / ticket)
│   └── 2026-06-14.md
│
├── protocol/               # working agreements
│   ├── handoff.md
│   ├── claim-lock.md
│   ├── branching.md
│   └── definition-of-done.md
│
└── compiled/               # output sinh tự động (KHÔNG sửa tay)
    └── .manifest.json      # hash nguồn → phát hiện stale
```

Compiler ghi ra **vị trí native** mà mỗi agent mong đợi (ngoài `.hatch/`):

```
repo/
├── CLAUDE.md               ← compiled cho Claude Code
├── AGENTS.md               ← compiled cho Codex
├── .kiro/steering/*.md     ← compiled cho Kiro
└── <antigravity-config>    ← compiled cho Antigravity
```

## Quan hệ với overclaud

overclaud cung cấp **lý thuyết layering + token-optimization + templates** cho riêng Claude. Trong Hatch:

- overclaud trở thành **một backend của compiler** — phần sinh ra `CLAUDE.md`/`.claude/` tái dùng templates và nguyên tắc token của overclaud.
- Các nguyên tắc "mỗi từ phải đáng giá token", "đúng scope đúng nơi" được nâng từ *1 agent × N surface* lên *N agent × M surface*.

## Trạng thái = nguồn sự thật phân tán nhưng hội tụ

Hatch không có database. **Hệ thống file + git LÀ database**:
- Vị trí thư mục của ticket (`backlog/` vs `in-progress/`) = trạng thái.
- Frontmatter ticket = metadata.
- Ledger = lịch sử bất biến.
- git = transaction log + cơ chế lock tự nhiên (xem [protocol](03-coordination-protocol.md)).

Lợi ích: con người đọc được, agent đọc được, diff được, review được qua PR, không cần hạ tầng thêm.
