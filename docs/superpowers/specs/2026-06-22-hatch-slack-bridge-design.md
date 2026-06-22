# Hatch Slack Bridge — design spec

_Status: approved (forks confirmed 2026-06-22) · Phase: GĐ2 / spec Phase E (thay
"Slack-like web UI" bằng **Slack thật**)_

## 1. Mục tiêu

Boss (user) mở **Claude Code** trong workspace, start thêm **Codex CLI** và
**Antigravity CLI (`agy`)**. Cả squad chat peer-to-peer qua Hatch bus (MCP).
Bridge này phơi toàn bộ chat đó lên **một Slack channel `#squad`** để boss:

1. **Xem** mọi tin agent nói với nhau (mỗi agent hiện đúng danh tính).
2. **`@tag` thẳng một agent từ Slack** để bơm prompt — daemon đánh thức đúng
   phiên có trí nhớ của agent đó.

Bất biến (từ spec gốc): *GĐ2 = GĐ1 với con người dời từ "ngoài, qua proxy" vào
"trong phòng, trực tiếp". Lõi không đổi.* Bridge **không** điều phối việc, không
đụng daemon — nó chỉ là cổng vào/ra Slack, đọc/ghi bus.

## 2. Quyết định đã chốt

| Nhánh | Chọn | Lý do |
|---|---|---|
| Scope v1 | **Bidirectional** (OUT + IN) | Toàn bộ giá trị GĐ2 là boss tag được agent. |
| Topology | **Một `#squad` + threads** | 1 task = 1 bus channel = 1 Slack thread; giống team người dùng 1 channel + thread theo việc. |
| Connection | **Socket Mode** | Chạy sau NAT trên máy Mac, không cần public URL/ngrok. Cần app-token `xapp-` + bot-token `xoxb-`. |
| Identity | **1 bot, impersonate** | `chat.postMessage` override `username`+`icon_emoji` → mỗi agent hiện là chính nó. Không cần N Slack app. |
| Lib | **github.com/slack-go/slack** | Có sẵn Web API + `socketmode` (handshake/ack/reconnect). Pin version, `go mod tidy`. |

## 3. Kiến trúc

```
bus (.hatch/bus, SoT)  ◄── MCP chat_post ── Claude/Codex/agy
   ▲          │
   │ ghi      │ tail (poll 2s)
   │          ▼
 ┌──────────────── hatch slack (bridge) ─────────────────┐
 │ OUT: bus→Slack  (impersonate username+icon, threadmap)│──► #squad
 │ IN : Slack→bus  (Socket Mode WS, chỉ tin người thật)  │◄── boss @tag
 └───────────────────────────────────────────────────────┘
   │ tin boss ghi vào bus (From=boss, Kind=user)
   ▼
 hatch daemon (riêng) ── resume-exec ──► đánh thức agent được @tag
```

Bridge và daemon là **hai process độc lập** cùng tail bus. Daemon không biết
Slack tồn tại; nó chỉ thấy một tin mới từ member boss và đánh thức người được tag.

## 4. OUT — bus → Slack

Tail bus theo cursor TS (đúng `daemon.tail()`). Với mỗi tin mới `m` ở channel `ch`:

1. **Bỏ qua nếu `m.From == cfg.Boss`** — boss đã gõ tin đó trong Slack rồi; đây
   cũng là chốt chống loop chiều IN.
2. Tra `threadmap[ch]`:
   - **Chưa có** → post `m` làm **root message** mới vào `#squad` (text prefix
     `*#<ch>*\n`), lưu `threadmap[ch] = ts` trả về.
   - **Đã có** → post `m` làm **reply** với `thread_ts = threadmap[ch]`.
3. Mỗi post override `username = displayName(m.From)`, `icon_emoji = icon(kind)`.

Bus có sub-threading (`InReplyTo`) nhưng Slack chỉ 1 cấp → **collapse**: cả
channel bus = 1 Slack thread; `InReplyTo` vẫn còn nguyên trong SoT, chỉ không
phản ánh vào layout Slack. Escalation (`From="hatch"` ở `dm-hatch-<boss>`) được
mirror bình thường → boss thấy cảnh báo.

## 5. IN — Slack → bus (Socket Mode)

Lắng `message` event. **Chỉ nuốt tin người thật**: bỏ qua khi có `bot_id`,
`user` rỗng, có `subtype` (edit/join), hoặc không thuộc `cfg.ChannelID`.

- **Reply trong thread** (`thread_ts` có): `ch = reverseThreadmap[thread_ts]`.
  Tìm thấy → post bus vào `ch`. Không thấy → tạo channel `t-<safe(thread_ts)>`.
- **Top-level** (không `thread_ts`): tạo channel mới `t-<safe(ts)>`,
  **set `threadmap[ch] = ts` ngay** (để reply OUT của agent nest dưới tin boss).

Post bus: `{Channel: ch, From: cfg.Boss, Type: msg, Body: text}`. `bus.Post` tự
rút `@codex`/`@claude` từ text (đã có `Mentions()`), điền `To` → daemon thấy →
resume. Boss gõ literal `@codex` (agent là impersonated username, **không** có
Slack user-id để `<@mention>` kiểu Slack — literal text là đúng đường).

## 6. Chống loop (bất biến an toàn)

- Tin agent → OUT mirror lên Slack (as agent) → Slack echo event có
  `bot_id`=bot mình → IN bỏ qua. ✔
- Tin boss trong Slack → IN ghi bus (`From=boss`) → OUT bỏ qua (`From==boss`). ✔

Không cần tag origin riêng; hai luật trên đủ.

## 7. Cấu phần & file

```
internal/slack/
  config.go     Config{BotToken,AppToken,ChannelID,Boss}; load env→.hatch/slack/config.json; validate
  identity.go   displayName(member) + icon(kind) map
  threadmap.go  load/save .hatch/slack/threadmap.json (ch→ts) + reverse in-mem
  bridge.go     Bridge{Bus,Roster,poster,Cfg,map,cursor}: mirrorOnce() (OUT), handleIncoming() (IN-pure)
  bridge_test.go  fake poster + temp bus: test OUT threadmap/skip-boss, IN top-level/reply/echo-skip
internal/cli/slackcmd.go   hatch slack [--interval 2s] [--once]; wires socketmode → handleIncoming, ticker → mirrorOnce
internal/paths/paths.go    SlackDir="slack", SlackConfig, SlackThreadmap
.gitignore                 .hatch/slack/
```

**Testability:** Slack client ẩn sau interface `poster{ Post(threadTS,user,icon,text) (ts,error) }`. OUT (`mirrorOnce`) và IN (`handleIncoming(channelID,user,botID,threadTS,ts,text)`) là method thuần, test bằng fake poster + temp bus. Socketmode chỉ là wiring mỏng trong cmd, gọi `handleIncoming`.

## 8. Config & bảo mật

- Token do **boss tự tạo** (Slack app: scopes `chat:write`,
  `chat:write.customize` (để override username+icon), `connections:write`; bật
  Socket Mode). Nạp qua env `HATCH_SLACK_{BOT_TOKEN,APP_TOKEN,CHANNEL,BOSS}`
  hoặc `.hatch/slack/config.json`.
- **Không hardcode, không commit token.** `.hatch/slack/` vào `.gitignore`.
- Đây là Slack workspace của chính boss, chính boss yêu cầu → không vướng rule
  "comm ra ngoài chưa review".

## 9. Giới hạn v1 (ngoài scope)

- `@agy`/`@kiro` từ Slack: ghi vào bus nhưng daemon **không** auto-resume (chưa
  có headless contract — spec gốc đã ghi). Agent thấy qua MCP inbox khi tới lượt,
  hoặc boss chuyển sang terminal đó.
- OUT poll 2s (không fsnotify) — chấp nhận, đơn giản.
- Slack 1 cấp thread — `InReplyTo` bus không phản ánh vào Slack layout.
- Chạy `hatch slack` + `hatch daemon` là **hai lệnh**. `hatch up` gộp = sau.

## 10. DoD

- [ ] `make lint` xanh; `go test ./...` và `-tags hatch_legacy` đều xanh.
- [ ] Build sạch cả hai tag.
- [ ] `hatch slack --once` chạy OUT một nhịp không cần Socket Mode (smoke).
- [ ] Token không lọt vào git; `.hatch/slack/` gitignored.
- [ ] @tag reviewer khác, không tự merge (human gate).
