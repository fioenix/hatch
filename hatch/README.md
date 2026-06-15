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
| `kb/` | **Wiki / bộ não chung của đội** (Confluence) — đọc *và* ghi |
| `workflow.yaml` | Quy trình làm việc của đội — template, sửa được |
| `board/` | Bảng sprint (Jira) |
| `ledger/` | Sổ standup / nhật ký hoạt động / git log |
| `protocol/` | Working agreements của đội |
| `compiler` | Bộ sinh tài liệu onboarding cho từng người mới |
| `orchestrator` | Engineering Manager / Scrum Master |

> **Bộ nhớ chung.** Agents không chia sẻ RAM (mỗi con một process), nhưng cùng khai thác và đóng góp vào **Knowledge Base** (`kb/`) — y như một đội người không đọc được não nhau nhưng cùng tra cứu và cập nhật một wiki. KB là bộ nhớ chung *bền* của hệ; agent vừa *input* (tra cứu để khỏi suy diễn lại) vừa *ghi lại* (quyết định, bài học, gotcha) khi làm. Xem [09-knowledge-base](docs/09-knowledge-base.md).

## Bốn agent, bốn vai mặc định

| Agent | Vai mặc định | Vì sao |
|---|---|---|
| **Claude Code** (chính) | Architect / Tech Lead + **Conductor** | Reasoning sâu, giữ bức tranh tổng thể, lập kế hoạch |
| **Kiro** | Spec-driven Implementer | Mạnh quy trình PRD → design → tasks |
| **Codex** | Autonomous Implementer | Thực thi nhanh, lặp nhiều |
| **Antigravity CLI** | Implementer / Utility | Linh hoạt, gánh việc phụ |

Bảng trên **chỉ là template khởi đầu**. Vai trò là do **người dùng tự cấu hình ở từng project** qua `registry.yaml` của project đó — không fix cứng. Cùng một agent có thể giữ vai khác nhau ở các project khác nhau. Xem [02-roles](docs/02-roles.md).

## Bảy trụ thiết kế

1. **Single Source of Truth → compile xuống từng agent.** Một nguồn canonical (`context/`), một compiler sinh ra `CLAUDE.md` / `AGENTS.md` / `.kiro/steering/` … tự động. Không copy-paste, không drift.
2. **Knowledge Base dùng chung** (`kb/`) — bộ nhớ chung bền của hệ; mọi agent đọc *và* ghi. Thay cho "shared memory" mà các process không có.
3. **Token optimization = context phân tầng** (L0 mission → L1 role → L2 task + KB on-demand). Mỗi agent chỉ nạp tầng của nó + ticket đang làm.
4. **Role assignment per-project** — map vai ↔ điểm mạnh từng agent; user tự thiết kế ở mỗi project.
5. **Coordination protocol** — điều phối qua artifact trong repo (board + ledger), async, không agent nào gọi trực tiếp agent khác.
6. **Workflow = template sửa được** — mặc định Agile ghép spec-driven (`Charter → Spec → Backlog → Sprint → In-Progress → Review → Done`); user có quyền thiết kế lại hoàn toàn ở mỗi project qua `workflow.yaml`.
7. **Governance & audit** — ledger append-only, gate trước merge, giới hạn thẩm quyền agent.

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
| 04 | [context-compiler](docs/04-context-compiler.md) | SSOT vs KB, compile per-agent, 3 tầng token |
| 05 | [workflow](docs/05-workflow.md) | Workflow-as-template, lifecycle, ceremonies |
| 06 | [governance](docs/06-governance.md) | Ledger, gates, giới hạn thẩm quyền |
| 07 | [orchestrator](docs/07-orchestrator.md) | Phase 3: CLI `hatch` launch & drive agent |
| 08 | [roadmap](docs/08-roadmap.md) | Lộ trình Phase 1 → 2 → 3 |
| 09 | [knowledge-base](docs/09-knowledge-base.md) | KB dùng chung: cấu trúc, đọc/ghi, vs SSOT/ledger |
| 10 | [agent-adapters](docs/10-agent-adapters.md) | Compile surface + headless invocation từng agent CLI |
| 11 | [communication](docs/11-communication.md) | Agents nói chuyện trực tiếp: DM · ask/reply · convene |
| 12 | [ceremonies-escalation](docs/12-ceremonies-escalation.md) | Standup · retro · planning · escalation · decision→ADR |
| 13 | [management](docs/13-management.md) | (thiết kế) Workload · performance · budget/lương — góc CEO/CTO |
| 14 | [org-and-cadence](docs/14-org-and-cadence.md) | (thiết kế) Org-chart/uỷ quyền · external deps · heartbeat |
| 15 | [obsidian-kb](docs/15-obsidian-kb.md) | (thiết kế) Obsidian vault làm KB chính qua CLI |
| 16 | [document-templates](docs/16-document-templates.md) | (thiết kế) Template/spec tài liệu theo framework, custom được |
| 17 | [pre-implementation](docs/17-pre-implementation.md) | Quyết định cần chốt trước khi implement toàn bộ |
| 18 | [observability](docs/18-observability.md) | Quan sát agents: transcript · TUI · tmux/Zellij |
| 19 | [going-real](docs/19-going-real.md) | Chuyển từ mock sang agent CLI thật (doctor, creds, an toàn) |

Sơ đồ trực quan: [overview](docs/overview.md) (bản đồ tổng plaintext) · [architecture-diagram](docs/architecture-diagram.md) (kiến trúc + workflow + sequence).

Spec kỹ thuật: [registry](spec/registry.schema.md) · [ticket](spec/ticket.schema.md) · [ledger](spec/ledger.schema.md) · [workflow](spec/workflow.schema.md)

## CLI (`hatch`)

Hatch là CLI viết bằng Go (single binary). Phase 1+2 đã chạy được:

```bash
./scripts/onboard.sh         # build + dựng demo workspace (mock agent) để thử ngay — hoặc `make onboard`
./scripts/onboard.sh --install  # thêm: go install hatch + hatch-mock lên PATH
# hoặc thủ công:
make build                  # → bin/hatch
bin/hatch init -w scrum      # 8 template: scrum kanban spec-first lite dual-track shape-up stage-gate incident
bin/hatch compile            # SSOT → CLAUDE.md / AGENTS.md / GEMINI.md / .kiro/steering
bin/hatch compile --check    # CI: fail nếu output stale so với SSOT
bin/hatch validate           # kiểm tra registry + workflow + board
bin/hatch ticket new --title "Export CSV" --role implementer --priority P1
bin/hatch ticket claim T-001 --agent codex --why "sprint S1"
bin/hatch ticket move T-001 --to review --by implementer --agent codex \
    --why "code xong" --handoff "endpoint ở export.go"
bin/hatch gate T-001 --to review     # đánh giá gate mà không di chuyển
bin/hatch status                     # board + cảnh báo WIP
bin/hatch standup --days 1           # digest ledger
bin/hatch kb add --type decision --title "CSV streaming" --tags export
bin/hatch kb query export
# Phase 3 — orchestrator (spawn agent headless theo docs/10)
bin/hatch run T-001 --claim --dry-run   # build invocation; bỏ --dry-run để chạy thật
bin/hatch plan --dry-run                 # spawn Conductor bẻ việc
bin/hatch watch --dry-run --max 3        # gán + chạy backlog (tôn trọng WIP)
bin/hatch board                          # mission-control TUI 4 pane: board + live output + ledger + chat
bin/hatch chat                           # (tuỳ chọn) TUI chat-only kiểu Slack
# Giao tiếp trực tiếp giữa agent (Slack-style — docs/11)
bin/hatch msg --from codex --to '#design,reviewer' --channel '#design' "Streaming hay buffer?"
bin/hatch inbox claude-code --mark       # message gửi tới mình (id + vai + #channel + *)
bin/hatch ask --from codex --to claude-code "Chốt giúp: streaming chứ?"   # hỏi-đáp đồng bộ
bin/hatch convene --topic "Thiết kế export API" --agents claude-code,codex,kiro --rounds 2
bin/hatch channel ls                     # liệt kê channel/DM/thread
# Nghi thức squad (docs/12)
bin/hatch ceremony standup               # digest theo agent + blockers → #standup
bin/hatch ceremony retro --write         # tổng kết chu kỳ + ứng viên đề bạt KB→SSOT
bin/hatch escalate T-001 --why "kẹt gate 2 lần"   # gọi senior/on-call (auto khi gate fail ≥2)
bin/hatch pair T-001 --driver codex --navigator claude-code --rounds 3   # pair programming
bin/hatch mob T-001 --agents codex,claude-code,gemini    # mob (driver xoay vòng)
bin/hatch ceremony demo · grooming       # sprint review · backlog refinement
bin/hatch presence set kiro --status offline --note PTO  # availability → phân việc theo capacity
bin/hatch oncall set --rotation claude-code,codex        # lịch trực; escalation nhắm người trực
```

## Trạng thái implement

- **Phase 1+2 — xong:** model + filesystem store, `init` (scaffold + 4 workflow template), compiler đa surface (Claude/Codex/Gemini/Kiro) + manifest stale-detection, workflow engine (transition + gate + no-self-review + dependency), ledger append-only, Knowledge Base, các lệnh `status/standup/validate/gate/ticket/kb`.
- **Phase 3 — xong (cơ chế):** orchestrator + adapter cho từng agent (`hatch run`/`plan`/`watch`) dựng invocation headless đúng theo [adapters](docs/10-agent-adapters.md), worktree isolation, TUI dashboard (`hatch board`). Adapter `kiro`/`antigravity`/`manual` rơi về handoff khi không có headless. `--dry-run` cho phép kiểm tra invocation mà không cần cài agent.

Có unit + integration test, CI, Makefile. Mã nguồn: `cmd/hatch` + `internal/`. Thiết kế gốc trong `docs/`. Kiến trúc (Lean Hexagonal, ports & adapters): [ARCHITECTURE.md](ARCHITECTURE.md).
