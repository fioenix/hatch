# 21 — Test drive: hai agent thật trò chuyện qua Hatch

Cách chạy thử **thật** để thấy nhiều coding-agent phối hợp qua chat dùng chung.

## Mô hình (đọc trước — khác hình dung "tự spawn")

Hatch là **embedded harness**, KHÔNG tự mở terminal/spawn agent (mô hình tự-lái
đã archive sau `hatch_legacy`). Luồng thật:

1. **Bạn** mở mỗi agent trong một terminal riêng (mỗi agent một session).
2. Mỗi session đã được wire vào **một chat dùng chung** (qua MCP) + có hook
   brief-on-start.
3. **Claude (orchestrator)** mở thread cho task, `@tag` giao việc; agent khác đọc
   inbox, làm, rồi trả lời **trong thread**.
4. Bạn xem toàn bộ qua `hatch chat` / `hatch board`.

Đây là phối hợp turn-based do bạn điều nhịp (prompt từng session), không phải tự
động hoàn toàn — nhưng state (chat/backlog/KB) là **dùng chung thật**.

## Tiền đề

- `hatch` trên PATH; `hatch setup` đã chạy (wire codex); `hatch init` trong repo.
- ≥2 agent đã đăng nhập. Kiểm: `hatch doctor` — cột AUTH + MCP/HOOK phải ✓.
  Bộ chắc ăn nhất: **Claude + Codex** (thường đã authed sẵn).
- Branch chưa push → Claude nạp plugin bằng `--plugin-dir` (không dùng marketplace).

## Smoke test (30 giây, không cần agent)

Xác nhận chat/backlog chạy trước khi mở agent thật:

```bash
cd <repo>
hatch msg --from "human:me" --channel "#smoke" "@codex ping"
hatch inbox codex          # phải thấy tin trên
hatch thread "#smoke"      # phải in lại hội thoại
rm -f .hatch/bus/threads/smoke.md   # dọn
```

## Live test — 3 terminal

Mở 3 terminal, đều `cd <repo>` (vd `/Users/fioenix/Projects/overclaud`).

### Terminal 3 — màn hình theo dõi (mở trước)

```bash
hatch chat        # Slack-style, read-only. (hoặc `hatch board`)
```

### Terminal 1 — Claude Code (orchestrator)

```bash
claude --plugin-dir <repo>/hatch/plugin
```

Khi vào, hook `SessionStart` chạy `hatch brief` → Claude thấy inbox + thread mở.
Dán prompt:

> Mình là Conductor của squad Hatch trên repo này. Tạo một task nhỏ và giao cho
> Codex qua chat: **viết hàm Go `Reverse(s string) string` đảo ngược chuỗi
> Unicode-safe, kèm 1 test**. Dùng tool `chat_open` mở thread cho task này,
> `@codex` trong nội dung, nêu acceptance rõ ràng. Sau đó cho mình biết đã mở
> thread nào.

→ Claude gọi `chat_open(@codex, …)`. Xem Terminal 3: thread mới xuất hiện.

### Terminal 2 — Codex (worker)

```bash
codex
```

Lần đầu Codex sẽ hỏi **TRUST** hook `hatch brief`/`hatch guard` — duyệt. Hook
brief chạy → Codex thấy task. Nếu không, dán prompt:

> Đọc inbox Hatch của bạn (tool `chat_inbox`), tìm task được giao, làm nó, rồi
> `chat_post` kết quả vào đúng thread, `@claude-code` để báo xong.

→ Codex gọi `chat_inbox` → thấy task → viết hàm + test → `chat_post` kết quả.
Xem Terminal 3: reply của Codex hiện trong thread.

### Terminal 1 — Claude review

> Check inbox/thread, xem Codex trả gì, review kết quả (đạt acceptance chưa?),
> rồi `chat_post` nhận xét vào thread.

## Kỳ vọng quan sát được (Terminal 3)

```
#reverse-... · claude-code → @codex   : **task** + acceptance
#reverse-... · codex → @claude-code   : kết quả + test
#reverse-... · claude-code → @codex   : review
```

- Mỗi agent **được brief** khi mở session (thấy việc đang chờ).
- Hai agent **dùng chung một thread** (state thật, không copy-paste).
- Thử cho Claude/Codex **sửa `.hatch/charter.md`** → `hatch guard` (PreToolUse)
  **chặn** (policy.protect).

## Kiểm chứng nhanh bằng CLI (song song)

```bash
hatch status                 # thread (task) + roster
hatch search reverse         # full-text qua chat
hatch thread "#reverse-..."  # toàn bộ một thread
```

## Trục trặc thường gặp

- **Claude không có tool hatch** → chưa nạp plugin; phải `claude --plugin-dir <repo>/hatch/plugin`.
- **Codex không chạy hook** → chưa TRUST; chạy lại Codex và duyệt, hoặc xem `~/.codex/hooks.json`.
- **`hatch brief` rỗng** → chưa có gì trong inbox/thread (bình thường khi mới bắt đầu).
- **Muốn dùng `/plugin install` thay `--plugin-dir`** → push branch lên GitHub trước.
- **Thêm agent thứ 3 (agy/kiro)** → cần đăng nhập (`agy`, `kiro-cli login`); agy mở
  bình thường, kiro chạy `kiro-cli --agent hatch`.
