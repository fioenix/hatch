# 10 — Agent Adapters (compile surfaces + headless invocation)

Tài liệu tham chiếu cho **compiler** (ghi instruction ra đâu) và **orchestrator** (spawn agent thế nào). Mọi thông tin dưới đây từ research docs chính thức tháng 6/2026; chỗ nào chưa chắc đều ghi rõ. Code trong `internal/compile` và `internal/orchestrator` cài đặt đúng theo bảng này.

## Bảng tổng hợp

| Agent | `kind` | Surface (compile ra) | Headless command |
|---|---|---|---|
| Claude Code | `claude` | `CLAUDE.md` (+ `.claude/agents/*.md`) | `claude -p` |
| Codex | `codex` | `AGENTS.md` | `codex exec` |
| Kiro | `kiro` | `.kiro/steering/*.md` | `kiro-cli chat --no-interactive` |
| Gemini CLI | `gemini` | `GEMINI.md` | `gemini -p` |
| Antigravity | `antigravity` | `AGENTS.md` (dùng chung convention) | IDE (chưa có CLI chính thức xác nhận) |
| (test) | `mock` | — | `hatch-mock` — agent giả để test end-to-end không cần CLI thật |
| (generic) | `manual` | — | không spawn; tạo handoff cho người/IDE |

`AGENTS.md` là convention dùng chung — **một** file phục vụ Codex + Antigravity (+ Gemini nếu khai `context.fileName`).

---

## Claude Code (`kind: claude`)

**Instruction file.** Đọc `CLAUDE.md` (project root) hoặc `.claude/CLAUDE.md`. **Không đọc `AGENTS.md`** — nếu cần thì `@AGENTS.md` import hoặc symlink. Nhiều `CLAUDE.md` theo cây thư mục được **nối** (root → cwd), không override. Import: `@path/to/file` (tối đa 4 hop, resolve theo file chứa import).

**Subagents.** `.claude/agents/<name>.md` — frontmatter: `name` (bắt buộc, lowercase-hyphen), `description` (bắt buộc), `tools` (CSV, kế thừa hết nếu bỏ), `model` (`sonnet|opus|haiku|fable|<id>|inherit`, mặc định `inherit`), `permissionMode`, `maxTurns`, `isolation: worktree`… Hatch map mỗi **role** → một subagent file (tùy chọn).

**settings.json.** `permissions.allow/deny/ask` (rule `Tool(pattern)`, ưu tiên deny→ask→allow), `hooks`. Precedence: managed > CLI flags > local > project > user.

**Headless.** `claude -p "<prompt>" [--bare] --output-format json|stream-json --permission-mode <m> --allowedTools "..." --model <m> --append-system-prompt-file <f> --max-turns N`. stdin được nạp làm context. JSON out có `result`, `session_id`, `total_cost_usd`; `--json-schema` → `structured_output`. CI nên dùng `--bare` (bỏ auto-discovery) + `ANTHROPIC_API_KEY`.
> permission-mode: `default|acceptEdits|plan|auto|dontAsk|bypassPermissions`. `dontAsk` = khóa chặt cho CI.

## Codex (`kind: codex`)

**Instruction file.** `AGENTS.md` (cũng `AGENTS.override.md` ưu tiên hơn). Merge root → cwd, cap mặc định **32 KiB** (`project_doc_max_bytes`). Global `~/.codex/AGENTS.md` **tùy version** — bản mới có thể không auto-merge; an toàn hơn dùng `developer_instructions` trong `config.toml`.

**Config.** `~/.codex/config.toml` (TOML): `model`, `model_provider`, `approval_policy` (`untrusted|on-failure|on-request|never`), `sandbox_mode` (`read-only|workspace-write|danger-full-access`), `[sandbox_workspace_write]` (`writable_roots`, `network_access=false`), `[mcp_servers.*]`.

**Headless.** `codex exec "<prompt>"` (alias `codex e`); prompt từ stdin nếu là `-`. Cờ: `-s workspace-write` (capability), `-m <model>`, `--json` (JSONL events), `-o <file>` (final message), `-C <dir>`, `--skip-git-repo-check`, `--ephemeral`, `-c key=value`. Resume: `codex exec resume --last "..."`.
> Trong `exec`, `-a/--ask-for-approval` và `--full-auto` không còn ý nghĩa (deprecated) — điều khiển quyền bằng `--sandbox`.

## Kiro (`kind: kiro`)

**Steering.** `.kiro/steering/*.md` (workspace) hoặc `~/.kiro/steering/*.md` (global). Mặc định: `product.md`, `tech.md`, `structure.md`. Frontmatter `inclusion`:
- `always` (mặc định) — nạp mọi lúc.
- `fileMatch` + `fileMatchPattern` (glob/array) — nạp khi đụng file khớp.
- `manual` — nạp khi `#tên-file` được nhắc.
> CLI có thể dùng `inclusion: auto` + `description` (skill-style) thay cho `always/fileMatch` — verify theo surface (IDE vs CLI).

**Specs.** `.kiro/specs/<feature>/{requirements.md, design.md, tasks.md}`. Requirements viết theo **EARS** (`WHEN … THE SYSTEM SHALL …`). Khớp khối spec-driven của Hatch.

**MCP.** `.kiro/settings/mcp.json` (workspace) / `~/.kiro/settings/mcp.json`.

**Headless.** `kiro-cli chat --no-interactive "<prompt>"`; auth qua env `KIRO_API_KEY` (bỏ qua login). Có thể pipe context: `git diff | kiro-cli chat --no-interactive "review"`. Không có user-input giữa chừng. (Kiro CLI 2.0+.)

## Gemini CLI (`kind: gemini`)

**Instruction file.** `GEMINI.md` (global `~/.gemini/GEMINI.md` + project + nested, most-specific wins). Import `@path`. Có thể đọc `AGENTS.md` qua `settings.json`: `"context": { "fileName": ["AGENTS.md","GEMINI.md"] }`.

**Config.** `.gemini/settings.json` (project) > `~/.gemini/settings.json` (user) > system. `mcpServers` block hoặc `gemini mcp add`.

**Headless.** `gemini -p "<prompt>" -m <auto|pro|flash|flash-lite> --approval-mode <default|auto_edit|yolo|plan>` (hoặc `-y/--yolo`), `--include-directories`, `-s/--sandbox`, `-o/--output-format text|json|stream-json`. JSON: `{response, stats, error?}`.

## Antigravity (`kind: antigravity`)

IDE agent-first (Agent Manager; surfaces editor/terminal/browser) ra mắt ~11/2025, quanh Gemini 3. Đọc `GEMINI.md` + `AGENTS.md` + thư mục rules workspace (**`.agent/rules/`** — chưa chắc) + Skills. Global rules `~/.gemini/AGENTS.md`.
> **Chưa xác nhận:** một CLI headless `agy` (`agy -p`) chỉ thấy ở blog thứ cấp 2026, không có nguồn chính thức. Vì vậy Hatch coi Antigravity là **surface AGENTS.md** + adapter `manual` (handoff cho IDE) cho tới khi có CLI chính thức.

---

## Hệ quả thiết kế cho Hatch

1. **Surface dedup:** compile theo *surface file*, không theo agent. Codex + Antigravity cùng `AGENTS.md` → sinh một lần. Mỗi agent trong registry khai `surfaces: [...]`; compiler hợp nhất.
2. **Layering giữ nguyên:** mọi surface nhận L0 (charter) + L1 (role(s) của agent đọc surface đó) + **con trỏ** tới L2. Không nhồi `context/` đầy đủ vào surface.
3. **Adapter orchestrator** (Phase 3) map `kind` → command + cờ ở bảng trên; `manual`/`kiro`(nếu thiếu key)/`antigravity` rơi về chế độ handoff.
4. **Sandbox/permission** dịch từ workflow gate + registry policy sang cờ tương ứng mỗi agent (vd `--permission-mode dontAsk` ↔ `--sandbox read-only`).
