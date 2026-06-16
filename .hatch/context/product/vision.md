# Product context — Hatch (L2)

## Cho ai
Người dùng chạy **một squad nhiều coding-agent** (Claude Code, Codex, agy, Kiro)
trên cùng một repo và cần chúng phối hợp thay vì giẫm chân nhau.

## Vấn đề
Mỗi agent CLI là một silo: không thấy việc của nhau, không có backlog chung,
tri thức không tích luỹ. Điều phối thủ công qua copy-paste không scale.

## Giải pháp
Hatch là **embedded harness**: agent tự lái, Hatch lo hạ tầng chung và phơi qua MCP.
- **Chat = giao tiếp + backlog**: mỗi thread là một task; `@tag` để gọi đồng đội;
  `chat_inbox` để "đọc phòng" trước khi vào việc.
- **Knowledge Base**: ghi/đọc quyết định, tra cứu trước khi làm lại.
- **Orchestrator**: một agent (mặc định Claude Code) giữ ghế conductor, điều phối
  team **qua chat** — chọn bằng `hatch init --client`.

## Onboarding 2 tầng
- `hatch setup` (1 lần/máy): global `~/.hatch` + wire CLI (codex/agy/plugin).
- `hatch init` (mỗi repo): `.hatch` local + chọn orchestrator + compile surfaces.

## Không làm
- Không thay thế agent CLI; không tự sinh code. Không là CI/CD hay issue tracker
  bên ngoài — backlog sống trong chat của repo.
