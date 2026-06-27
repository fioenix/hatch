# 01 — Architecture

## Bảy trụ

```
                    ┌─────────────────────────────┐
                    │     ORCHESTRATOR (hatch)    │  ← Phase 3: EM/Scrum Master
                    │  plan · run · gate · status │
                    └──────────────┬──────────────┘
                                   │ đọc/ghi
        ┌──────────────────────────┼──────────────────────────┐
        ▼                          ▼                           ▼
 ┌────────────┐           ┌────────────────┐          ┌────────────────┐
 │  CONTEXT   │  compile  │     BOARD      │  ghi vào │     LEDGER     │
 │  (SSOT)    │ ────────► │   (tickets)    │ ───────► │ (audit append) │
 └─────┬──────┘           └───────┬────────┘          └────────────────┘
       │                          │ claim/execute              ▲
       ▼ sinh ra                  ▼                            │ events
 ┌──────────────────────┐   ┌──────────────────────────────────┐
 │ CLAUDE.md / AGENTS.md │   │  AGENTS theo ROLES (registry)    │
 │ .kiro/steering/ …     │◄──│  Architect·Implementer·Reviewer  │
 └──────────────────────┘   └───────────────┬──────────────────┘
        ▲                       đọc ▲        │ ghi (learnings/ADR)
        │                           │        ▼
        │                    ┌──────┴────────────────────┐
        │                    │  KNOWLEDGE BASE (kb/)      │  ← bộ nhớ chung, đọc+ghi
        │                    └────────────────────────────┘
        └──────────── PROTOCOL + WORKFLOW (working agreements) ───────────┘
```

1. **Context (SSOT)** — nguồn canonical duy nhất, *đầu vào* compile (con người/Architect chủ yếu ghi).
2. **Knowledge Base** (`kb/`) — bộ nhớ chung; agent đọc *và* ghi. SSOT là "config vào", KB là "tri thức vào-ra", Ledger là "sự kiện ra" (xem [09-knowledge-base](09-knowledge-base.md)).
3. **Compiler** — sinh file instruction native cho từng agent từ SSOT.
4. **Roles + Registry** — định nghĩa vai và gán agent vào vai (cấu hình per-project).
5. **Board + Workflow** — hàng đợi ticket; lane/transition do `workflow.yaml` định nghĩa (template sửa được).
6. **Ledger + Protocol** — sổ audit append-only + quy ước phối hợp (claim/lock, handoff, branching, DoD).
7. **Orchestrator** — (Phase 3) CLI điều khiển toàn bộ.

### Ba kho tri thức — đừng lẫn

| Kho | Hướng | Ai ghi | Ví dụ |
|---|---|---|---|
| `context/` (SSOT) | vào (compile → agent) | Human / Architect | conventions, PRD, stack |
| `kb/` (Knowledge Base) | vào **và** ra | Mọi agent | ADR, bài học, gotcha, ghi chú domain |
| `ledger/` | ra (sự kiện) | Mọi agent | "ai claim/làm/gate cái gì, vì sao" |

## Layout `.hatch/` trong repo đích

```
.hatch/
├── charter.md              # L0 — mission chung, nhỏ gọn (mọi agent đều nạp)
├── registry.yaml           # roster agent + năng lực + binding vai (per-project)
├── workflow.yaml           # quy trình: lane, transition, gate, ceremony (template, sửa được)
│
├── roles/                  # L1 — context theo vai, mỗi vai 1 file (user định nghĩa)
│   ├── architect.md
│   ├── implementer.md
│   ├── reviewer.md
│   ├── tester.md
│   └── ...
│
├── context/                # SSOT — tri thức canonical (config), compile ra ngoài
│   ├── product/            #   PRD, domain, business rules
│   ├── tech/               #   stack, conventions, kiến trúc
│   └── shared.md           #   những gì mọi vai cần
│
├── kb/                     # KNOWLEDGE BASE — bộ nhớ chung, agent đọc+ghi
│   ├── decisions/          #   ADR — quyết định kiến trúc + lý do
│   ├── domain/             #   tri thức nghiệp vụ tích lũy
│   ├── learnings/          #   bài học, gotcha, pitfall đã gặp
│   ├── index.md            #   mục lục để tra cứu nhanh (tiết kiệm token)
│   └── .meta.json          #   tag/owner/updated cho từng mục
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

## Nền tảng: layering + token optimization

Compiler dựa trên **lý thuyết layering + token-optimization + templates** cho Claude:

- Phần sinh ra `CLAUDE.md`/`.claude/` là **một backend của compiler**, tái dùng các template và nguyên tắc token đó.
- Các nguyên tắc "mỗi từ phải đáng giá token", "đúng scope đúng nơi" được nâng từ *1 agent × N surface* lên *N agent × M surface*.

## Trạng thái = nguồn sự thật phân tán nhưng hội tụ

Hatch không có database. **Hệ thống file + git LÀ database**:
- Vị trí thư mục của ticket (`backlog/` vs `in-progress/`) = trạng thái.
- Frontmatter ticket = metadata.
- Ledger = lịch sử bất biến.
- git = transaction log + cơ chế lock tự nhiên (xem [protocol](03-coordination-protocol.md)).

Lợi ích: con người đọc được, agent đọc được, diff được, review được qua PR, không cần hạ tầng thêm.
