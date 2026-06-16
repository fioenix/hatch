# Charter (L0 — mission)

> Đây là tầng L0: ngắn gọn, mọi agent nạp mỗi message. Giữ dưới ~300 token.
> Sửa file này là sửa SSOT — chạy `hatch compile` để đẩy xuống mọi agent.

## Sản phẩm
**Hatch** — embedded harness cho một squad coding-agent làm chung trên một repo.
Coding agent là entrypoint và **tự lái**; Hatch cung cấp **chat dùng chung**
(= giao tiếp + backlog, mỗi thread = một task) và **Knowledge Base**, phơi cho
agent qua một **MCP server**. CLI Go single-binary; dùng được với Claude Code,
Codex, Antigravity (`agy`), Kiro.

## Nguyên tắc tối cao
- **Embedded, không điều khiển**: Hatch KHÔNG spawn/lái agent. `board`/`chat`/
  `status` chỉ là view read-only. Quy trình là **protocol compile thành prose**
  (CLAUDE.md/AGENTS.md/…), không phải engine cưỡng chế.
- **SSOT → compile**: sửa `.hatch/{charter,roles,registry,workflow}` rồi
  `hatch compile`; không sửa file output.
- **Minimum code, surgical**: giải đúng vấn đề, không abstraction thừa. Lean
  Hexagonal (model / port / adapter).
- Mỗi thay đổi có dấu vết (ledger) + lý do (`why`); không sửa ngoài scope ticket.

## Ràng buộc
- Go 1.24+. Build sạch **cả hai**: mặc định và `-tags hatch_legacy`.
- **Không sửa file ngoài repo**: config home của Codex/agy do `hatch setup` lo;
  `hatch doctor` chỉ gọi lệnh auth của CLI, KHÔNG quét thư mục creds.
- `make lint && make test` (+ legacy) phải pass trước khi push; `go mod tidy` nếu đổi deps.
