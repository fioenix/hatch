# Shared context (SSOT)

Điều mọi vai cần biết. Đây là nguồn canonical — compile xuống agent, không sửa file output.

## Glossary
- **SSOT** — `.hatch/{charter,roles,registry,workflow}`: nguồn sự thật, compile xuống surfaces.
- **Surface** — file protocol compile ra cho từng agent: `CLAUDE.md`, `AGENTS.md`,
  `GEMINI.md`, `.kiro/steering/`. Commit bình thường; KHÔNG sửa tay (regenerate qua compile).
- **Thread = task** — một channel chat là một đơn vị công việc; message gốc = id task.
- **Orchestrator / conductor** — agent giữ ghế điều phối (mặc định `claude-code`),
  chọn qua `hatch init --client`; surface của nó nhận "khối orchestrator".
- **MCP** — cách agent với tới chat + KB: `hatch mcp --as <id>` (stdio).
- **Local vs global** — `.hatch` trong repo đè `~/.hatch` global (resolve local trước).

## Quy ước chung
- Ngôn ngữ: tiếng Việt cho thảo luận; thuật ngữ kỹ thuật giữ tiếng Anh.
- Commit: bắt đầu bằng động từ (Add/Fix/Refactor/…), atomic, kèm `why`.
- Giao tiếp qua chat: mở thread cho mỗi task, `@tag` đúng agent/role, brief lại trong thread.
- DoD: code + test pass (cả `hatch_legacy`) + lint, trước khi xin merge (human gate).
