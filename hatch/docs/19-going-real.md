# 19 — Từ mock sang chạy thật

Hướng dẫn chuyển một workspace từ **mock agent** (demo) sang **agent CLI thật**.

## 1. Cài agent CLI (không bắt buộc hết — chỉ cần ≥1)
Cài cái nào bạn dùng (xem [10-agent-adapters](10-agent-adapters.md)):
- **Claude Code** — `claude` ([code.claude.com](https://code.claude.com))
- **Codex** — `codex` ([github.com/openai/codex](https://github.com/openai/codex))
- **Antigravity CLI** — `agy` (kế nhiệm Gemini CLI; `agy` ([antigravity.google](https://antigravity.google)))
- **Kiro** — `kiro-cli` ([kiro.dev](https://kiro.dev))

## 2. Đặt `kind` thật trong registry
Trong `.hatch/registry.yaml`, đổi `kind: mock` về `claude|codex|gemini|kiro`, khai thêm:
```yaml
agents:
  - id: codex
    kind: codex
    roles: [implementer, tester]
    model: gpt-5.2          # tùy chọn
    sandbox: workspace-write # codex: read-only|workspace-write|danger-full-access
  - id: claude-code
    kind: claude
    roles: [conductor, architect, reviewer]
    approval: acceptEdits    # claude permission-mode
    budget_usd: 200          # track (xem 13)
    rate_per_mtok: 15
```

## 3. Đăng nhập (OAuth) hoặc cấp env key
Ưu tiên **login OAuth** của từng CLI (không cần key trong repo/env):
```bash
claude            # /login (subscription) — hoặc env ANTHROPIC_API_KEY
codex login       # ChatGPT OAuth — hoặc env OPENAI_API_KEY
agy               # lần đầu: chọn Google OAuth (browser/device-code) — hoặc env ANTIGRAVITY_API_KEY
kiro-cli          # login — headless cần env KIRO_API_KEY
```
Token do từng CLI tự quản (OS keyring / file riêng) — Hatch **không** đụng vào. Nếu tự động hoá thì cấp env key tương ứng (không bao giờ commit vào repo).

## 4. Kiểm tra sẵn sàng
```bash
hatch doctor      # config hợp lệ? compiled fresh? CLI nào đã cài? credential nào có?
hatch compile     # nếu doctor báo stale
```
`hatch doctor` xanh hết = sẵn sàng.

## 5. Chạy + quan sát
```bash
hatch run T-001 --dry-run                 # xem invocation sẽ chạy (an toàn)
hatch run T-001 --claim                    # chạy thật, claim trước
hatch logs T-001 -f                        # tail live output
hatch board                                # TUI: board + live + activity
hatch run T-001 --mux=tmux                 # mỗi run một pane tmux/zellij
hatch watch --parallel 3                   # tự phân việc backlog, chạy song song
hatch tick                                 # một nhịp cho cron/CI
```

## 6. An toàn khi chạy thật
- **Sandbox bảo thủ** mặc định (codex `workspace-write`, claude `acceptEdits`); nâng quyền phải khai rõ.
- **Gate `human-merge`** giữ con người ở vòng cuối trước khi vào `done`.
- **`no-self-review`**: reviewer ≠ implementer (đã cưỡng chế).
- **Budget**: track-only — theo dõi qua `hatch budget`/`report`; chưa auto-pause (quyết định [17](17-pre-implementation.md)).
- **Mọi thứ auditable**: ledger + transcript (`.hatch/runs/`) + bus đều trên git, review qua PR.

## 7. Quay lại mock bất cứ lúc nào
`./scripts/onboard.sh` dựng demo mock; hoặc đổi một agent về `kind: mock` để thử luồng mà không tốn token.
