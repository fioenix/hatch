# 21 — Manual test case: squad nhiều agent thật trên Hatch

Bài test thủ công end-to-end: nhiều coding-agent phối hợp qua chat dùng chung,
và quan sát chính Hatch bằng `hatch trace`. Làm theo từng bước, tick ☐ khi đạt.

---

## 0. Mô hình (đọc trước — tránh hiểu nhầm)

Hatch là **embedded harness**, **KHÔNG tự spawn agent** (mô hình tự-lái đã archive).
Luồng thật:

- **Bạn** mở mỗi agent trong một terminal riêng.
- Mỗi session đã được wire vào **một chat dùng chung** (qua MCP) + hook brief-on-start.
- **Claude (orchestrator)** mở thread cho task, `@tag` giao việc; agent khác đọc
  inbox, làm, trả lời **trong cùng thread**.
- Bạn xem qua `hatch chat` và soi chính Hatch qua `hatch trace`.

Đây là phối hợp **turn-based** do bạn điều nhịp (prompt từng session). State
(chat/backlog/KB) là dùng chung thật.

Dùng **Claude + Codex** (thường đã đăng nhập). agy/Kiro cần login thêm.

---

## 1. Pre-flight (1 lần)

```bash
cd /Users/fioenix/Projects/hatch      # hoặc repo của bạn
hatch doctor
```

PASS khi:
- ☐ `claude` và `codex`: cột **AUTH** = ✓
- ☐ `codex`: **MCP** ✓ và **HOOK** ✓
- ☐ dòng cuối `✓ ready`

Nếu codex chưa ✓ MCP/HOOK: `hatch setup --client codex`.
Nếu chưa có workspace trong repo: `hatch init`.

---

## 2. Bố trí 3 terminal

Cả 3 đều `cd` vào repo trước. Vai trò:

| Terminal | Lệnh khởi động | Vai |
|---|---|---|
| **T3** (mở trước) | `hatch chat` | màn hình theo dõi (read-only; thoát `q`) |
| **T1** | `claude --plugin-dir $PWD/hatch/plugin` | orchestrator |
| **T2** | `codex` | worker |

> Branch chưa push → Claude phải nạp plugin bằng `--plugin-dir`. Khi đã push thì
> thay bằng `/plugin marketplace add fioenix/hatch` + `/plugin install hatch@hatch`.

---

## 3. T1 — Claude mở task và giao cho Codex

Trong T1 (sau khi Claude khởi động; nếu hỏi approve MCP server `hatch` → đồng ý),
dán prompt:

> Bạn là Conductor của squad Hatch. Dùng tool MCP `hatch`:
> 1. `whoami` xác nhận bạn là claude-code.
> 2. `chat_open`: title "Reverse string", giao **@codex** viết hàm Go
>    `Reverse(s string) string` đảo chuỗi Unicode-safe + 1 test; nêu acceptance.
> 3. Cho tôi biết đã mở channel tên gì.

PASS khi:
- ☐ Claude gọi `whoami` → `claude-code`
- ☐ Claude gọi `chat_open` → trả về channel (vd `#reverse-string`)
- ☐ **T3** hiện thread mới: `claude-code → @codex`

---

## 4. T2 — Codex đọc việc, làm, trả lời

Khởi động `codex`. Lần đầu nó hỏi **TRUST hook** (`hatch brief`, `hatch guard`)
và/hoặc MCP server `hatch` → **đồng ý hết**. Dán prompt:

> Bạn là codex trong squad Hatch. Dùng tool MCP `hatch`:
> 1. `chat_inbox` xem việc đang chờ bạn.
> 2. Làm task (viết hàm + test).
> 3. `chat_post` kết quả vào ĐÚNG thread, `@claude-code` báo xong.

PASS khi:
- ☐ Codex gọi `chat_inbox` → thấy task "Reverse string"
- ☐ Codex `chat_post` kết quả vào đúng thread
- ☐ **T3** hiện: `codex → @claude-code` trong cùng thread

---

## 5. T1 — Claude review

Dán vào T1:

> Gọi `chat_inbox` rồi `chat_read` thread vừa rồi. Đánh giá kết quả Codex có đạt
> acceptance không, rồi `chat_post` nhận xét vào thread.

PASS khi:
- ☐ Claude thấy reply của Codex (chung 1 thread, không copy-paste)
- ☐ **T3** hiện lượt 3: `claude-code → @codex`

→ Bạn vừa thấy đủ vòng **giao việc → làm → phản hồi** giữa 2 agent thật.

---

## 6. Test guard (governance enforcement)

Dán vào T1 (hoặc T2):

> Sửa file `.hatch/charter.md`, thêm một dòng bất kỳ.

PASS khi:
- ☐ Tool edit bị **chặn** với lý do kiểu "policy: .hatch/charter.md is protected"
  (đây là `hatch guard` chạy ở PreToolUse)

---

## 7. Observability — soi chính Hatch

Ở terminal bất kỳ (cùng repo):

```bash
hatch trace                 # mọi tool-call: ai · tool · ✓/✗ · latency
hatch trace --errors        # chỉ call lỗi (= issue của Hatch để fix)
hatch doctor                # cuối bảng: "● MCP: N tool-call lỗi gần đây" nếu có
```

PASS khi:
- ☐ `hatch trace` liệt kê các call của claude-code + codex từ bước 3–6
- ☐ Nếu có lỗi, `hatch trace --errors` chỉ ra đúng tool + thông điệp lỗi

> Mẹo: chạy `hatch trace --follow` ở T3 thay cho `hatch chat` nếu muốn nhìn
> tool-call realtime thay vì hội thoại.

---

## 8. Đối chiếu bằng CLI (tuỳ chọn)

```bash
hatch status                      # thread (task) + roster
hatch thread "#reverse-string"    # toàn bộ một thread
hatch search reverse              # full-text qua chat
```

---

## 9. Dọn sau test

```bash
# Xoá thread + log test (đều là runtime, đã gitignore — không ảnh hưởng git):
rm -f .hatch/bus/threads/reverse-string.md .hatch/logs/mcp.jsonl
# Nếu agent có tạo file code demo trong repo, xoá thủ công.
```

---

## Troubleshooting

| Triệu chứng | Xử lý |
|---|---|
| Claude không có tool `hatch_*` | Quên `--plugin-dir`; thoát, mở lại đúng lệnh ở bước 2 |
| Codex báo hook/MCP chưa trust | Duyệt lại trong Codex; kiểm `hatch doctor` (codex MCP/HOOK = ✓) |
| `hatch chat` / `hatch trace` trống | Chưa agent nào gọi tool; làm bước 3 trước |
| Guard không chặn | `hatch doctor` xem HOOK; guard fail-open nên nếu agent dùng tool lạ sẽ allow |
| Muốn thêm agent 3 (agy/kiro) | Đăng nhập (`agy`, `kiro-cli login`); kiro chạy `kiro-cli --agent hatch` |

---

## Phụ lục — test tự động (headless, không cần mở terminal tay)

Để verify nhanh bằng LLM thật (CI/dev), drive agent ở chế độ headless. **Cách tin
cậy nhất:** dùng `claude` làm LLM driver cho bất kỳ identity nào.

```bash
# 1) Tạo mcp config trỏ identity muốn đóng vai:
echo '{"mcpServers":{"hatch":{"command":"hatch","args":["mcp","--as","codex"]}}}' > /tmp/as-codex.json

# 2) Worker (identity=codex) đọc inbox + trả lời — KHÔNG bypass quyền, chỉ allow hatch tools:
claude -p "Call chat_inbox, then chat_post to '#tc-reverse' a Reverse one-liner and @claude-code." \
  --mcp-config /tmp/as-codex.json --strict-mcp-config \
  --allowedTools "mcp__hatch__chat_inbox,mcp__hatch__chat_post,mcp__hatch__chat_read"

# 3) Orchestrator (claude-code, qua plugin) mở task / review:
claude -p "<prompt>" --plugin-dir "$PWD/hatch/plugin" \
  --allowedTools "mcp__plugin_hatch_hatch__chat_open,mcp__plugin_hatch_hatch__chat_inbox,mcp__plugin_hatch_hatch__chat_read,mcp__plugin_hatch_hatch__chat_post"

# 4) Quan sát: hatch trace   (và hatch thread "#tc-reverse")
```

**Tool-name namespacing cho `--allowedTools`:**
- Qua plugin (`--plugin-dir`): `mcp__plugin_hatch_hatch__<tool>`
- Qua `--mcp-config`: `mcp__hatch__<tool>`

**Lưu ý headless theo agent (đã đo):**
- `codex exec` gọi MCP tool hay treo headless → dùng claude-as-codex ở trên thay thế.
- `kiro-cli chat --no-interactive` load MCP **flaky**; `--agent hatch` cần file ở
  `.kiro/agents/hatch.json` (init đã ghi đúng).
- `agy -p` cần `--dangerously-skip-permissions` (không có allowlist scoped).
- Interactive (mở terminal tay) thì các quirk trên **không xảy ra** — đó là cách
  manual test chính ở trên.
