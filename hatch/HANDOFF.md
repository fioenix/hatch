# HANDOFF — tiếp tục phát triển Hatch ở local

Tài liệu bàn giao để dev tiếp Hatch trên máy của bạn. Đọc cùng:
`docs/20-embedded-harness-pivot.md` (mô hình hiện hành), `ARCHITECTURE.md`
(Lean Hexagonal), `docs/architecture-diagram.md` (sơ đồ).

## 1. Hatch là gì (sau pivot)

Hatch là **embedded harness** cho một squad coding-agent làm trên một repo:
**coding agent là entrypoint và tự lái**; Hatch cung cấp **chat dùng chung
(= giao tiếp + backlog, thread = task)** và **Knowledge Base**, phơi cho agent
qua **MCP server**. Hatch **không** spawn/điều khiển agent. `hatch board`/
`chat`/`status` chỉ là **view read-only**. Quy trình/phân vai là **protocol
được compile** vào `CLAUDE.md`/`AGENTS.md`/`GEMINI.md`/`.kiro` (prose), không
phải engine cưỡng chế.

> Docs `00–19` mô tả thiết kế **trước pivot** (Hatch tự lái). Khi mâu thuẫn,
> **doc 20 thắng** và hành vi CLI thực tế là chuẩn.

## 2. Build / test / run

Yêu cầu: **Go 1.24+** (go.mod khai `go 1.25.0`), **git**.

```bash
cd hatch
make build           # → bin/hatch (+ bin/hatch-mock, chỉ dùng cho legacy)
make lint            # go vet + kiểm gofmt
make test            # toàn bộ test (build mặc định)
make install         # go install ./cmd/hatch → $(go env GOPATH)/bin
./scripts/onboard.sh # build + demo trong ./demo-workspace (không cần agent thật)
```

Hai cấu hình build — **luôn chạy cả hai trước khi push**:

```bash
go build ./...                 && go test ./...                 # default (sản phẩm)
go build -tags hatch_legacy ./... && go test -tags hatch_legacy ./...  # + operator đã archive
```

## 3. Trạng thái hiện tại (đã xong)

- **MCP server** `hatch mcp --as <agent>` (stdio) — tools: whoami, chat_open,
  chat_post, chat_read, chat_inbox, chat_search, chat_channels, kb_add,
  kb_search. `--as` rỗng → `$HATCH_AGENT` → agent kind=claude đầu tiên.
  (`internal/mcpserver/`, `internal/cli/mcp.go`)
- **compile** tiêm protocol (workflow prose + chat etiquette + DoD self-check
  + khối orchestrator cho lead) vào surfaces; đăng ký MCP per-agent.
  (`internal/compile/render.go`, `mcp.go`)
- **Claude plugin** `hatch/plugin/` (MCP + skill `hatch-chat` + `/hatch`) +
  `.claude-plugin/marketplace.json` ở repo root.
- **board/chat/status read-only** (`internal/tui/`, `internal/cli/status.go`).
- **`hatch setup`** — onboarding máy 1 lần: tạo global `~/.hatch` + wire client
  home-scoped (codex `~/.codex`, agy `~/.gemini`) + plugin Claude. Interactive khi
  có TTY, hoặc `--client a,b --yes` cho CI. (`internal/cli/setup.go`)
- **`hatch init [--client cc|codex|agy|kiro]`** — chạy trong repo: tạo `.hatch`
  **local** (mặc định), chọn 1 client làm **orchestrator** (mặc định cc → ghi
  `orchestrator: <id>` vào registry.yaml giữ nguyên comment), compile, wire agent
  project-scoped (claude `.mcp.json`, kiro `.kiro/`). `--global` để nhắm `~/.hatch`.
  (`internal/cli/init.go`, `client.go`; lead resolve ở `compile/bundle.go`)
- **Workspace phân tầng**: `~/.hatch` global + `.hatch` local override; output
  compile luôn vào repo hiện tại. (`internal/paths/`, `config.Workspace.Out()`)
- **Operator tự-lái archived** sau tag `hatch_legacy` (run/plan/watch/tick,
  orchestrator, workflow-engine, ceremonies, ask/convene, pair/mob, presence,
  oncall, cost/budget, workload/perf, report).

## 4. CẦN VERIFY Ở LOCAL (chưa kiểm được trên remote)

Đây là việc đầu tiên nên làm — môi trường remote không có agent CLI thật:

1. **MCP handshake thật** với từng agent:
   - Claude Code: cài plugin (`/plugin marketplace add fioenix/overclaud` →
     `/plugin install hatch@hatch`) hoặc dựa `.mcp.json`; gọi thử tool
     `whoami`, `chat_open`, `chat_inbox`. Kiểm 2 agent (vd claude + codex) cùng
     thấy một chat: agent A `chat_open @codex …`, agent B `chat_inbox` thấy.
   - Codex: `hatch init --client codex` (cần `codex` trên PATH → nó gọi
     `codex mcp add hatch -- hatch mcp --as codex`). Xác nhận `~/.codex/
     config.toml` có `[mcp_servers.hatch]` và Codex gọi được tool.
   - **agy (Antigravity CLI)**: `hatch init --client agy` ghi
     `~/.gemini/config/mcp_config.json`. **Cần xác nhận runtime path đúng** với
     bản agy bạn cài (có migration; xem `google-antigravity/antigravity-cli#60`).
     Lưu ý bug stdout non-TTY (#76) có thể ảnh hưởng.
   - Kiro: `.kiro/settings/mcp.json`.
2. **`hatch doctor`** với agent thật đã đăng nhập (chỉ gọi lệnh auth của từng
   CLI, KHÔNG quét thư mục creds — giữ nguyên nguyên tắc này).
3. **Orchestrator ≠ Claude**: ✅ cơ chế hoá + verify local — `hatch init
   --client codex` ghi `orchestrator: codex`, compile đặt khối orchestrator vào
   AGENTS.md (rời khỏi CLAUDE.md). Còn cần xác nhận agent đó **thật sự điều phối
   qua chat** với CLI thật.

## 5. Kiến trúc & nơi sửa

Lean Hexagonal (xem `ARCHITECTURE.md`):
- `internal/model/` — domain types (Message, KBEntry, Registry, Workflow…).
- `internal/port/` — interfaces (Board/Ledger/Bus/KB…).
- Adapters: `internal/bus/` (chat), `internal/store/` (kb/ledger/board),
  `internal/compile/` (SSOT→surfaces+MCP), `internal/mcpserver/` (MCP).
- Driving: `internal/cli/` (Cobra), `internal/tui/` (Bubble Tea), `cmd/hatch/`.
- SSOT mẫu + template: `internal/scaffold/templates/`.

Thêm một MCP tool: sửa `internal/mcpserver/server.go` (+ struct ở `types.go`).
Đổi nội dung compiled: `internal/compile/render.go`. Thêm client: `resolveClientKind`
+ `setupClient` trong `internal/cli/client.go`.

## 6. Hạn chế & ý tưởng kế tiếp

- **agy path** dựa trên tài liệu, chưa chạy binary thật → verify (#4).
- **compile manifest** lưu trong SSOT; khi global SSOT dùng cho NHIỀU repo,
  `compile --check` chỉ nhớ outputs repo cuối — chấp nhận tạm; nếu cần, đưa
  manifest về theo output-root.
- **Plugin** mới có cho Claude; Codex/Kiro/agy dựa surfaces (đủ dùng).
- `cmd/hatch-mock` chỉ phục vụ legacy orchestrator — cân nhắc gỡ khỏi
  `make build` mặc định.
- Ý tưởng: `hatch doctor` đọc/thử `~/.gemini/config/mcp_config.json` &
  `~/.codex/config.toml` để báo trạng thái đăng ký MCP per-client; pixel-game
  visualize cho `hatch chat` (đã nêu trong tầm nhìn).

## 7. Git

- Branch: `claude/agents-control-tower-bo6w3f` (PR #1, draft).
- Quy ước: dev trên branch này; chạy `make lint && make test` (+ legacy) trước
  khi push; `go mod tidy` nếu đổi deps (CI có check tidy).
