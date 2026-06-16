# Tech context — Hatch (L2)

## Stack
- **Go single-binary** (Go 1.24+), Cobra (CLI), Bubble Tea (TUI read-only),
  `modelcontextprotocol/go-sdk` (MCP server stdio).
- **Lean Hexagonal**: `internal/model` (domain types) · `internal/port`
  (interfaces) · adapters `internal/bus` (chat) · `internal/store` (kb/ledger/board)
  · `internal/compile` (SSOT→surfaces+MCP) · `internal/mcpserver` (MCP).
  Driving: `internal/cli` (Cobra) · `internal/tui` · `cmd/hatch`.

## Build
- `make build` → `bin/hatch`. `make install` → `$(go env GOPATH)/bin`.
- Luôn build/test **cả hai** trước khi push:
  `go build ./... && go test ./...` và `... -tags hatch_legacy`.
- Operator tự-lái cũ archived sau tag `hatch_legacy` (run/plan/watch, orchestrator,
  workflow-engine, ceremonies, …). Không thêm tính năng vào nhánh legacy.

## Quy ước
- MCP tool mới: `internal/mcpserver/server.go` (+ struct `types.go`).
- Đổi nội dung compiled: `internal/compile/render.go`. Thêm client: `client.go`.
- SSOT mẫu + template scaffold: `internal/scaffold/templates/`.
- Comment giải thích **WHY**, không phải WHAT. Pin deps, commit lock files.
