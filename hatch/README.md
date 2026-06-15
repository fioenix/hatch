# Hatch

> **Bản nâng cấp đa-agent của [overclaud](../README.md)** — sản phẩm thuộc hệ sinh thái **Finolabs** (Fioenix + Dinosaur Labs).
>
> overclaud (hiện tại) tối ưu instructions cho **một** agent trên nhiều surface. Hatch nâng chính overclaud lên để tối ưu instructions + context + **phối hợp** cho **nhiều** agent, trên nhiều surface, cùng một repo. overclaud cũ không bị thay thế — nó trở thành **compiler backend cho Claude** bên trong Hatch.

## Tên gọi

**Hatch** = ổ trứng *nở* ra thành viên đội. Đặt tên theo chủ đề khủng long/phượng hoàng của Finolabs, và khớp xương sống thiết kế — một bầy phối hợp săn mồi. (Nghĩa kép cũ "`hatch run <ticket>` nở một agent" thuộc về mô hình orchestrator trước pivot, nay đã archived — xem [doc 20](docs/20-embedded-harness-pivot.md).)

## Nó là gì

Hatch là một **embedded harness** cho coding agent. **Coding agent là entrypoint** — user mở Claude Code / Codex / Antigravity CLI (`agy`) / Kiro ngay trong workspace, và agent **tự lái mình**. Hatch không spawn, không điều khiển agent; nó cung cấp một **lớp nền chung** mà mọi agent với tới qua **MCP server**:

- một **chat dùng chung** (bus) — đồng thời là **backlog**: một thread = một task;
- một **Knowledge Base dùng chung** (bộ nhớ chung bền của hệ);
- một **ledger** audit append-only.

Nhờ đó nhiều agent trên cùng một repo phối hợp theo phong cách **async kiểu Slack** (mở thread cho mỗi task, brief tiến độ trong thread, `@mention` đồng đội, `chat_search`/`kb_search` để nhớ lại), giữ vai trò khác nhau theo một quy trình kiểu Agile, để lại audit trail đầy đủ, và **không lãng phí token** vì mỗi agent chỉ nạp đúng phần context của mình.

> **Lưu ý về docs.** Các doc **00–19 mô tả thiết kế GỐC (trước pivot)** — khi Hatch còn là orchestrator tự lái agent. Mô hình **hiện hành** là [doc 20 — embedded-harness pivot](docs/20-embedded-harness-pivot.md). Khi đối chiếu hành vi CLI thực tế, **doc 20 thắng**.

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
| `chat` (bus) | **Slack của đội** — kênh trao đổi *và* backlog (mỗi thread = một task) |
| `compiler` | Bộ sinh tài liệu onboarding + working agreements cho từng người mới |
| MCP server | Cánh cửa mỗi người dùng để vào Slack chung + wiki chung |
| agent lead/Conductor | Engineering Manager / Scrum Master — *là một agent tự lái*, không phải Hatch |

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

1. **Single Source of Truth → compile xuống từng agent.** Một nguồn canonical (`.hatch/`: `charter.md`, `roles/`, `registry.yaml`, `workflow.yaml`), một compiler sinh ra `CLAUDE.md` / `AGENTS.md` / `GEMINI.md` / `.kiro/steering` tự động. Không copy-paste, không drift.
2. **Knowledge Base dùng chung** (`kb/`) — bộ nhớ chung bền của hệ; mọi agent đọc *và* ghi qua MCP. Thay cho "shared memory" mà các process không có.
3. **Token optimization = context phân tầng** (L0 mission → L1 role → L2 task + KB on-demand). Mỗi agent chỉ nạp tầng của nó + thread task đang làm.
4. **Role assignment per-project** — map vai ↔ điểm mạnh từng agent; user tự thiết kế ở mỗi project qua `registry.yaml`.
5. **Coordination protocol = chat async.** Phối hợp qua **chat dùng chung** (bus): mở thread cho mỗi task, brief tiến độ trong thread, `@mention` để nhờ. Không agent nào spawn hay gọi trực tiếp agent khác; Hatch cũng không. Audit qua ledger.
6. **Workflow = protocol được COMPILE, không phải engine.** `workflow.yaml` (Agile: scrum/kanban/spec-first/…) + roles + DoD vẫn là SSOT, nhưng compile biến chúng thành **văn bản hành vi** (prose) tiêm vào instruction surface từng agent. Agent đọc và *tự* tuân theo; Hatch không cưỡng chế bằng code.
7. **Governance & audit** — ledger append-only; Definition-of-Done là **self-check agent tự chạy**; thẩm quyền agent khai trong SSOT.

## Mô hình phối hợp: chat async, agent tự lái

- **Agent đầu tiên user mở** (thường Claude Code) mặc định là **Conductor/orchestrator** và có thể kiêm các vai khác (architect/reviewer…). Việc này khai trong `registry.yaml` và tiêm qua compile vào CLAUDE.md (kèm khối "orchestrator").
- Conductor bẻ việc → **mở một thread cho mỗi task** trong chat, brief, `@tag` agent phù hợp.
- Agent được tag đọc thread, làm, brief kết quả lại **trong cùng thread**. Trạng thái task suy ra từ hội thoại (post type `done`/`block`/`decision`), không cần lane/claim/gate engine.
- Reviewer ≠ implementer tự chạy DoD self-check rồi báo done trong thread.
- Tất cả **async** (kiểu Slack): agent chỉ phản hồi khi nó đang chạy. Không lời gọi trực tiếp giữa agent, không ai spawn ai — y như một đội người trao đổi qua Slack.

## Đọc docs theo thứ tự

> **Quan trọng:** Doc **00–19 mô tả thiết kế GỐC trước pivot** (Hatch tự lái agent, orchestrator là entrypoint, board là control panel, backlog kiểu Jira). Mô hình **hiện hành** nằm ở **doc 20**. Khi mâu thuẫn, **doc 20 thắng**. Đọc doc 20 trước để hiểu mô hình thực tế, rồi dùng 00–19 cho bối cảnh/ý tưởng nền.

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
| **20** | [**embedded-harness-pivot**](docs/20-embedded-harness-pivot.md) | **MÔ HÌNH HIỆN HÀNH** — Hatch là embedded harness (MCP + chat=backlog), agent tự lái; thắng khi mâu thuẫn |

Sơ đồ trực quan: [overview](docs/overview.md) (bản đồ tổng plaintext) · [architecture-diagram](docs/architecture-diagram.md) (kiến trúc + workflow + sequence).

Spec kỹ thuật: [registry](spec/registry.schema.md) · [ticket](spec/ticket.schema.md) · [ledger](spec/ledger.schema.md) · [workflow](spec/workflow.schema.md)

## CLI (`hatch`)

Hatch là CLI viết bằng Go (single binary). Đây là **embedded harness**: agent là entrypoint, Hatch lo SSOT/chat/KB và phơi chúng qua MCP. Bộ lệnh mặc định (xem `internal/cli/root.go`):

```bash
./scripts/onboard.sh         # build + dựng demo workspace để thử ngay — hoặc `make onboard`
make build                   # → bin/hatch
bin/hatch init -w scrum       # 8 template: scrum kanban spec-first lite dual-track shape-up stage-gate incident
bin/hatch compile             # SSOT → CLAUDE.md / AGENTS.md / GEMINI.md / .kiro/steering
                              #   (protocol prose: workflow + chat etiquette + DoD self-check + khối orchestrator
                              #    cho lead) + đăng ký MCP per-agent (.mcp.json · .kiro/settings/mcp.json merge;
                              #    snippet .hatch/mcp/* cho codex/agy)
bin/hatch compile --check     # CI: fail nếu output stale so với SSOT
bin/hatch validate            # kiểm tra registry + workflow
bin/hatch mcp --as claude-code  # MCP server (stdio) phơi tool chat + KB với danh tính agent này
                              #   (--as bỏ trống → $HATCH_AGENT, rồi agent kind=claude đầu tiên)
bin/hatch status              # read-only: tóm tắt thread chat (task) + roster agent
bin/hatch board               # read-only TUI: THREADS + CHAT + ledger ACTIVITY
bin/hatch chat                # read-only TUI: viewer hội thoại kiểu Slack
bin/hatch kb add --type decision --title "CSV streaming" --tags export
bin/hatch kb query export     # kb add|query|index|link|backlinks|graph|open
bin/hatch msg --from human --channel '#design' "Streaming hay buffer?"   # human inject vào chat
bin/hatch inbox claude-code   # message gửi tới một agent
bin/hatch thread <id>         # xem một thread (task)
bin/hatch channel ls          # channel ls|show|join|leave|members
bin/hatch search export       # full-text qua chat
bin/hatch doc ...             # doc templates · logs · org · sync · hook · doctor
```

Luồng dùng thực tế:

```
hatch init → hatch compile → mở coding agent NGAY TRONG workspace
   (CLAUDE.md + .mcp.json đã wire nó vào chat + KB chung) → agent tự lái, làm việc qua MCP
Người xem bằng hatch board / hatch chat / hatch status; chèn ý kiến bằng hatch msg.
```

> **Lệnh archived (tùy chọn).** Bộ operator tự-lái (`run`, `plan`, `watch`, `tick`, orchestrator, workflow-engine `gate`/`escalate`/`ticket`, `ceremony`/`standup`, `ask`/`convene`, `pair`/`mob`, `presence`, `oncall`, `cost`/`budget`, `workload`/`perf`, `report`) **không** nằm trong binary mặc định. Chúng được archive sau build tag và chỉ build khi cần: `go build -tags hatch_legacy`. Khôi phục được, không bị xóa.

## Trạng thái implement

Pivot **embedded-harness** đã implement (xem [doc 20](docs/20-embedded-harness-pivot.md)):

- **MCP server** (`hatch mcp --as <agent>`, stdio) trên bus/KB — tools whoami · chat_open · chat_post · chat_read · chat_inbox · chat_search · chat_channels · kb_add · kb_search (`internal/mcpserver`, `internal/cli/mcp.go`).
- **compile đổi mục đích**: tiêm protocol (charter + roles + workflow-prose + DoD self-check + chat etiquette + khối orchestrator cho lead) vào CLAUDE.md/AGENTS.md/GEMINI.md/.kiro, kèm đăng ký MCP per-agent (`.mcp.json`, `.kiro/settings/mcp.json` merge; snippet `.hatch/mcp/*` cho Codex/agy).
- **Claude plugin** tại `hatch/plugin/` (MCP + skill `hatch-chat` + slash `/hatch`); `.claude-plugin/marketplace.json` ở repo root.
- **board/chat/status read-only**: board TUI = THREADS + CHAT + ledger ACTIVITY; chat = viewer; status = tóm tắt thread + roster. Không còn run/claim/compose.
- **Operator tự-lái archived** sau build tag `hatch_legacy` (không vào binary mặc định, khôi phục được).
- Cả build mặc định **và** `-tags hatch_legacy` đều compile/vet/test green.

Có unit + integration test, CI, Makefile. Mã nguồn: `cmd/hatch` + `internal/`. Thiết kế gốc (trước pivot) trong `docs/00`–`19`; mô hình hiện hành ở `docs/20`. Kiến trúc (Lean Hexagonal, ports & adapters): [ARCHITECTURE.md](ARCHITECTURE.md).
