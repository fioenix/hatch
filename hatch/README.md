# Hatch

> **Bản nâng cấp đa-agent của [overclaud](../README.md)** — sản phẩm thuộc hệ sinh thái **Finolabs** (Fioenix + Dinosaur Labs).
>
> overclaud (hiện tại) tối ưu instructions cho **một** agent trên nhiều surface. Hatch nâng chính overclaud lên để tối ưu instructions + context + **điều phối** cho **nhiều** agent, trên nhiều surface, cùng một repo. overclaud cũ không bị thay thế — nó trở thành **compiler backend cho Claude** bên trong Hatch.

## Tên gọi

**Hatch** = ổ trứng *nở* ra thành viên đội. Nghĩa kép: orchestrator `hatch run <ticket>` chính là *nở một agent ra làm việc*. Đặt tên theo chủ đề khủng long/phượng hoàng của Finolabs, và khớp xương sống thiết kế — một bầy phối hợp săn mồi.

## Nó là gì

Hatch biến một repo thành **không gian làm việc đa-agent có điều phối**. Cùng một codebase, nhiều coding agent (Claude Code, Codex, Kiro, Antigravity CLI) tham gia với những vai trò khác nhau, làm việc theo một quy trình kiểu Agile, để lại audit trail đầy đủ, và **không lãng phí token** vì mỗi agent chỉ nạp đúng phần context của mình.

## Phép ẩn dụ neo: một đội Agile người

Mọi thành phần trong Hatch đều có một đối ứng trong một squad người. Đây là la bàn thiết kế — khi phân vân, hỏi "đội người làm việc này thế nào?".

| Thành phần Hatch | Đối ứng trong squad người |
|---|---|
| `charter.md` | Team charter / mục tiêu sản phẩm |
| `roles/` | Bản mô tả công việc (JD) từng vai |
| `registry.yaml` | Danh bạ nhân sự + năng lực + ai giữ vai gì |
| `board/` | Bảng sprint (Jira) |
| `ledger/` | Sổ standup / nhật ký hoạt động / git log |
| `protocol/` | Working agreements của đội |
| `compiler` | Bộ sinh tài liệu onboarding cho từng người mới |
| `orchestrator` | Engineering Manager / Scrum Master |

## Bốn agent, bốn vai mặc định

| Agent | Vai mặc định | Vì sao |
|---|---|---|
| **Claude Code** (chính) | Architect / Tech Lead + **Conductor** | Reasoning sâu, giữ bức tranh tổng thể, lập kế hoạch |
| **Kiro** | Spec-driven Implementer | Mạnh quy trình PRD → design → tasks |
| **Codex** | Autonomous Implementer | Thực thi nhanh, lặp nhiều |
| **Antigravity CLI** | Implementer / Utility | Linh hoạt, gánh việc phụ |

Vai là do **người dùng chỉ định** qua `registry.yaml`; bảng trên chỉ là mặc định gợi ý.

## Sáu trụ thiết kế

1. **Single Source of Truth → compile xuống từng agent.** Một nguồn canonical (`context/`), một compiler sinh ra `CLAUDE.md` / `AGENTS.md` / `.kiro/steering/` … tự động. Không copy-paste, không drift.
2. **Token optimization = context phân tầng** (L0 mission → L1 role → L2 task). Mỗi agent chỉ nạp tầng của nó + ticket đang làm.
3. **Role assignment** — map vai ↔ điểm mạnh từng agent.
4. **Coordination protocol** — điều phối qua artifact trong repo (board + ledger), async, không agent nào gọi trực tiếp agent khác.
5. **Workflow** — Agile ghép spec-driven: `Charter → Spec → Backlog → Sprint → In-Progress → Review → Done`.
6. **Governance & audit** — ledger append-only, gate trước merge, giới hạn thẩm quyền agent.

## Mô hình điều phối: Hybrid

- **Conductor** (CC) chạy planning: bẻ epic → ticket, gán vai/agent, đặt ưu tiên, ghi dependency.
- **Workers** tự **claim** ticket trong lane của mình, thực thi, để lại ledger, đẩy ticket sang review.
- **Reviewer** gate → done.
- Tất cả qua board + ledger. Không có lời gọi trực tiếp giữa agent — y như một đội người: manager lập kế hoạch, dev pull ticket, reviewer duyệt.

## Đọc docs theo thứ tự

| # | Doc | Nội dung |
|---|---|---|
| 00 | [vision](docs/00-vision.md) | Vấn đề, tầm nhìn, phép ẩn dụ squad |
| 01 | [architecture](docs/01-architecture.md) | 6 trụ, layout `.hatch/`, các thành phần |
| 02 | [roles](docs/02-roles.md) | Mô hình vai, map năng lực agent |
| 03 | [coordination-protocol](docs/03-coordination-protocol.md) | Hybrid, board, claim/lock, handoff, DoD |
| 04 | [context-compiler](docs/04-context-compiler.md) | SSOT → compile per-agent, 3 tầng token |
| 05 | [workflow](docs/05-workflow.md) | Lifecycle Agile + spec-driven, ceremonies |
| 06 | [governance](docs/06-governance.md) | Ledger, gates, giới hạn thẩm quyền |
| 07 | [orchestrator](docs/07-orchestrator.md) | Phase 3: CLI `hatch` launch & drive agent |
| 08 | [roadmap](docs/08-roadmap.md) | Lộ trình Phase 1 → 2 → 3 |

Spec kỹ thuật: [registry](spec/registry.schema.md) · [ticket](spec/ticket.schema.md) · [ledger](spec/ledger.schema.md)

## Trạng thái

Đây là **bộ thiết kế** (design docs). Chưa implement. Implement theo [roadmap](docs/08-roadmap.md) sau khi docs được chốt.
