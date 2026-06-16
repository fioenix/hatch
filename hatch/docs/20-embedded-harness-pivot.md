# 20 — Pivot: Hatch là embedded harness (chat = comms + backlog)

> **Đính chính hướng.** Bản build hiện tại coi Hatch là **orchestrator tự lái agent** và là **entrypoint** (CLI/TUI user mở đầu). Mô hình đúng ngược lại: **coding agent là entrypoint; agent tự lái; Hatch là lớp nền chung (chat + bộ nhớ) mà agent với tới qua MCP**. Doc này xác định cái sai và đề xuất nâng cấp.

## User flow đúng

1. User mở **một coding agent** (Claude Code / Codex / agy / Kiro) và bắt đầu session làm việc với nó — *đây* là entrypoint.
2. User mở thêm **`hatch chat`** để **xem** các agent nói chuyện (read-only; sau này visualize kiểu pixel-game).
3. **Chat vừa là kênh giao tiếp vừa là backlog management.** Một task → agent **mở một thread**, làm, brief kết quả vào thread; cần hỗ trợ thì **@tag** agent khác — agent được tag đọc thread để hiểu nhiệm vụ rồi phản hồi **trong cùng thread**; task cần nhiều/all agent → mở **topic + tag all**.
4. **`hatch board` = chỉ xem stats + hội thoại.** Không điều khiển gì.
5. **Hatch tích hợp VÀO agent** dưới dạng **MCP server (+ skill/plugin)** — một *embedded harness*. Hatch **không** phải first TUI/UI để user bắt đầu.

## Cái đã đi sai (thẳng thắn)

| Đã build (sai hướng) | Phải là |
|---|---|
| Hatch **tự lái** agent: `run/plan/watch/tick`, `pickAgent` theo capacity, spawn headless, `--mux` | Agent tự lái mình; Hatch là comms+memory chung agent với tới qua **MCP**. Hatch **không spawn** agent. |
| Hatch = CLI/TUI **entrypoint** user mở đầu | Coding agent là entrypoint; Hatch **nhúng** (MCP/skill/plugin) |
| `hatch board` là bảng điều khiển (`r`=run, claim) | Board = **read-only** stats + xem chat |
| Backlog kiểu Jira riêng (lanes/tickets/claim/gate engine) | **Thread chat CHÍNH LÀ task**; backlog = tập thread |
| ask/convene/pair/mob do orchestrator spawn agent đích | **Agent tự khởi xướng**: post + @tag trong thread; agent được tag trả lời khi nó đang chạy (async). Không spawn. |
| compile → instruction "role + workflow" cho operator | compile → **đăng ký Hatch MCP + tiêm "chat etiquette"** vào từng agent |

## Cái GIỮ được (lõi tốt, không phí)

- **bus** = chat (channel · thread · @mention · search · inbox) → **đây chính là lõi sản phẩm**, đã có sẵn.
- **KB** (bộ nhớ chung) · **ledger** (audit) · filesystem+git làm DB.
- Kiến trúc hexagonal (model/ports/adapters) — tái dùng nguyên.
- **compile/SSOT** đổi mục đích: tiêm "Hatch chat etiquette" + đăng ký MCP server vào surface từng agent (CLAUDE.md/AGENTS.md/GEMINI.md/.kiro + file MCP config).
- **`hatch chat` TUI** (quan sát) → tương lai pixel-game viz.

## Tích hợp: MCP vs Skill vs Plugin

| Hình thức | Phạm vi | Vai trò |
|---|---|---|
| **MCP server** *(chính)* | **Mọi agent** đều hỗ trợ: Claude `.mcp.json`/`claude mcp add`; Codex `config.toml [mcp_servers]`; agy MCP (`~/.gemini/config`); Kiro `.kiro/settings/mcp.json` | Một `hatch mcp` (stdio) phơi *tools* chat/KB → mọi agent dùng chung một chat. **Đây là embedded harness.** |
| **Skill** | Claude-specific (hành vi) | Dạy *etiquette*: mở thread mỗi task, brief kết quả, tag khi cần. Bản cross-agent = đoạn hướng dẫn compile vào CLAUDE.md/AGENTS.md/… |
| **Plugin** | Claude Code | Gói MCP + skill + slash command; DX đẹp cho Claude nhưng theo từng hệ. |

→ **Đề xuất:** **MCP server là trục tích hợp**; **compile tiêm etiquette** vào instruction native + **đăng ký MCP** trong config từng agent. (Plugin Claude là lớp đóng gói tùy chọn về sau.)

## MCP tool surface (đề xuất) — backed by bus/KB/ledger
- `chat.open_thread(channel, title, body)` → trả về thread/task id
- `chat.post(channel, body, reply_to?)` · `chat.reply(thread, body)`
- `chat.read(channel|thread)` · `chat.inbox()` · `chat.search(query)`
- `chat.threads(status?)` — liệt kê task (thread) đang mở
- `chat.tag(...)` (hoặc @mention trong body — đã hỗ trợ)
- `kb.add/kb.search/kb.link`
- `whoami()` — agent này là ai (gán per MCP-server instance qua `--as`)

Mỗi agent chạy MCP server với danh tính của nó (`hatch mcp --as claude-code`), nên `post/tag/inbox` tự gắn đúng "from".

## Thread = task (định nghĩa lại backlog)
- **Thread root = một task**; trạng thái suy ra từ hội thoại (open/in-progress/done/blocked) qua quy ước nhẹ (field `status` ở message gốc, hoặc post type `done`/`block`), **không** cần lane/claim/gate engine nặng.
- **Board = stats trên thread** (đang mở/đang chạy/blocked, ai đang phản hồi, decisions) — read-only.
- Workflow engine (lane/transition/gate/no-self-review) trở thành **overlay tùy chọn**, không phải mặc định.

## Lộ trình nâng cấp (phased)
1. **`hatch mcp`** (stdio MCP server) trên bus/KB sẵn có — năng lực mới cốt lõi.
2. **compile đổi mục đích**: ghi đăng ký MCP (`.mcp.json` · `~/.codex/config.toml` · `.kiro/settings/mcp.json` · agy MCP) + block "chat etiquette" vào từng surface.
3. **board read-only**: bỏ điều khiển run/claim khỏi TUI; chỉ stats + chat.
4. **Hạ cấp/bỏ orchestration tự lái**: `run/plan/watch/tick`, pickAgent/capacity, mux, pair/mob/convene-dạng-spawn. Pair/mob/convene **tái định nghĩa thành pattern chat** (skill dạy), không phải vòng lặp Hatch spawn.
5. **thread-as-task**: quy ước status + board stats từ thread; workflow engine thành tùy chọn.
6. Giữ KB/ledger/clean-arch.

## Quyết định đã chốt (2026-06-15)

- **Tích hợp: MCP server + Claude plugin.** `hatch mcp` (stdio) phơi tool chat/KB cho mọi agent (Claude/Codex/agy/Kiro); thêm gói **plugin Claude Code** (đăng ký MCP + skill etiquette + slash) cho DX. compile đăng ký MCP vào config từng agent.
- **Lean pivot ở RUNTIME.** Bỏ phần Hatch tự-lái: `run/plan/watch/tick`, `pickAgent`/capacity/presence, `--mux`, board-control, **Go workflow-engine enforce lane/gate/transition**, pair/mob/convene-dạng-spawn, cost/oncall-as-runtime. Hatch không spawn, không enforce bằng code.
- **Workflow + phân vai = PROTOCOL được COMPILE, không phải engine.** `registry.yaml` (ai giữ vai gì) + `workflow.yaml` (quy trình agile: scrum/kanban/…) + charter + roles + DoD **vẫn là SSOT**, nhưng compile biến chúng thành **văn bản hành vi** tiêm vào `CLAUDE.md`/`AGENTS.md`/`GEMINI.md`/`.kiro`. Agent đọc và *tự* tuân theo qua chat (mở thread, phân vai, gate = self-checklist), Hatch không cưỡng chế.
- **Agent đầu tiên = orchestrator + đa vai.** Agent user mở trước (vd Claude Code) mặc định là **Conductor/orchestrator** (chắc chắn) và **có thể kiêm các vai khác** (architect/reviewer…). Việc này khai trong registry + tiêm qua compile. Các agent khác tham gia async qua chat khi chúng đang chạy.

### Hệ quả: "agile workflow inject ở đâu?" → vào instruction surface

```
.hatch/ (SSOT)                      hatch compile (đổi mục đích)
  charter.md (mission)        ─┐
  roles/*.md (vai + ranh giới) ├─►  CLAUDE.md  (lead: "Bạn là Conductor; chạy scrum;
  registry.yaml (ai giữ vai)   │       mở 1 thread/task; phân vai; @tag; gate=DoD;
  workflow.yaml (agile process)│       dùng Hatch MCP chat …")  + đăng ký MCP server
  protocol/DoD                ─┘    AGENTS.md / GEMINI.md / .kiro  (tương tự theo vai)
                                    + .mcp.json / config.toml / .kiro/settings/mcp.json
```

Workflow templates (scrum/kanban/spec-first/…) **vẫn giữ**, nhưng từ "cấu hình cho engine" → **mô tả quy trình bằng prose** mà orchestrator-agent tuân theo. Gate (`make test`…) → mục trong **Definition-of-Done** mà agent tự chạy/tự kiểm, không phải Hatch chạy.

### Thread = task (chat = backlog)
- Agent mở **thread/topic cho mỗi task** qua MCP; brief tiến độ + kết quả vào thread; @tag để nhờ; tag-all cho việc chung.
- Trạng thái task suy ra từ hội thoại (post type `done`/`block`/`decision`), không lane-engine.
- `hatch board` = stats read-only trên thread + ledger; `hatch chat` = xem hội thoại (→ pixel viz sau).

### Lộ trình implement (đã chốt) — TRẠNG THÁI

1. ✅ **`hatch mcp --as <agent>`** (stdio MCP server) trên bus/KB — tools: whoami, chat_open, chat_post, chat_read, chat_inbox, chat_search, chat_channels, kb_add, kb_search. (`internal/mcpserver`, `internal/cli/mcp.go`). `--as` mặc định = `$HATCH_AGENT` hoặc agent kind=claude đầu tiên.
2. ✅ **compile đổi mục đích**: tiêm protocol (charter+roles+workflow-prose+DoD+chat-etiquette) vào CLAUDE.md/AGENTS.md/GEMINI.md/.kiro; khối "orchestrator" cho lead agent; đăng ký MCP cho kiro (`.kiro/settings/mcp.json` merge; snippet `.hatch/mcp/*` cho codex/agy). Claude nạp MCP qua plugin → không ghi `.mcp.json`. (`internal/compile/render.go`, `mcp.go`).
3. ✅ **Claude plugin** (`hatch/plugin/`): `.mcp.json` + skill `hatch-chat` + slash `/hatch`; `.claude-plugin/marketplace.json` ở repo root.
4. ✅ **board/chat read-only**: board TUI = THREADS + CHAT + ACTIVITY(ledger); chat TUI = viewer; `status` tóm tắt thread + roster. Bỏ run/claim/compose.
5. ✅ **Prune runtime operator → archive** sau build tag `hatch_legacy` (khôi phục được, không vào binary mặc định): run/plan/watch/tick, orchestrator, workflow-engine (gate/escalate/ticket), ceremonies, ask/convene, pair/mob, presence, oncall, cost/budget, workload/perf, report.
6. ✅ Giữ: bus · KB · ledger · clean arch · compile · registry/workflow/charter/roles/DoD as SSOT.

> **Lưu ý docs.** Các doc 00–19 mô tả **thiết kế gốc (trước pivot)** — Hatch tự lái agent. Mô hình hiện hành là doc này (20). Khi đối chiếu hành vi CLI thực tế, doc 20 thắng.
