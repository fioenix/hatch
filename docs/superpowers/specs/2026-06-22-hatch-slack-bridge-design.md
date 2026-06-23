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
| Connection | **Socket Mode** (trên hub app) | Chạy sau NAT trên máy Mac, không cần public URL/ngrok. Cần app-token `xapp-`. |
| Identity | **Mỗi agent một Slack app/bot thật** | Agent là Slack principal thật → `@mention`/DM/avatar/presence native. Một **hub app** chạy Socket Mode (inbound) + là giọng fallback cho escalation & agent chưa có token. |
| Lib | **github.com/slack-go/slack** | Có sẵn Web API + `socketmode` (handshake/ack/reconnect). Pin version, `go mod tidy`. |

> **Ghi chú quyết định:** đã cân nhắc 1-bot-impersonate (zero-setup nhưng không
> mention/DM native được vì agent không phải principal thật) vs mỗi-agent-một-app
> (fidelity cao, đúng "mô phỏng team người", nhưng tốn N app khi onboard). Chọn
> **mỗi agent một app**; agent nào thiếu token thì **fallback** sang hub bot +
> username override (degrade êm, không crash).

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
3. Chọn client theo `m.From`: agent có app riêng → post bằng **bot token của
   chính nó** (danh tính thật, name/avatar do app đó cấu hình). Không có →
   **fallback** hub bot + override `username=displayName(m.From)`,`icon_emoji`.

Bus có sub-threading (`InReplyTo`) nhưng Slack chỉ 1 cấp → **collapse**: cả
channel bus = 1 Slack thread; `InReplyTo` vẫn còn nguyên trong SoT, chỉ không
phản ánh vào layout Slack. Escalation (`From="hatch"` ở `dm-hatch-<boss>`) →
fallback hub bot (icon `:bell:`) → boss thấy cảnh báo.

## 5. IN — Slack → bus (Socket Mode, trên hub app)

Lắng `message` event. **Chỉ nuốt tin người thật**: bỏ qua khi có `bot_id`,
`user` rỗng, có `subtype` (edit/join), hoặc không thuộc `cfg.ChannelID`.

- **Reply trong thread** (`thread_ts` có): `ch = reverseThreadmap[thread_ts]`.
  Tìm thấy → post bus vào `ch`. Không thấy → tạo channel `t-<safe(thread_ts)>`.
- **Top-level** (không `thread_ts`): tạo channel mới `t-<safe(ts)>`,
  **set `threadmap[ch] = ts` ngay** (để reply OUT của agent nest dưới tin boss).

**Dịch mention:** boss `@codex` trong Slack → tin chứa `<@Ucodex>` (Slack
resolve sang bot user-id của app codex). `translateMentions` đổi `<@Ucodex>` →
`@codex` nhờ bảng `mentions[botUserID]=agentID` (dựng lúc start qua `auth.test`
mỗi agent token). Mention người thật (id không map) giữ nguyên token.

Post bus: `{Channel: ch, From: cfg.Boss, Type: msg, Body: text}`. `bus.Post` tự
rút `@codex` từ text (đã có `Mentions()`), điền `To` → daemon thấy → resume.

## 6. Chống loop (bất biến an toàn)

- Tin agent → OUT mirror lên Slack (as agent) → Slack echo event có
  `bot_id`=bot mình → IN bỏ qua. ✔
- Tin boss trong Slack → IN ghi bus (`From=boss`) → OUT bỏ qua (`From==boss`). ✔

Không cần tag origin riêng; hai luật trên đủ.

## 7. Cấu phần & file

```
internal/slack/
  config.go     Config{AppToken,HubToken,ChannelID,Boss,Agents{id→token}}; load json→env override; validate
  identity.go   displayName(member) + icon(kind) — dùng cho fallback impersonation
  threadmap.go  load/save .hatch/run/slack/threadmap.json (ch→ts) + reverse in-mem
  bridge.go     Bridge{Bus,Roster,poster,tm,mentions,cursor}: mirrorOnce() (OUT), handleIncoming()+translateMentions() (IN-pure)
  runtime.go    Run(): hub client + per-agent clients, auth.test→mentions map, socketmode loop, multiPoster/dryPoster
  bridge_test.go  fake poster + temp bus: OUT threadmap/skip-boss/from, IN top-level/reply/echo-skip/mention-translate
internal/cli/slackcmd.go   hatch slack [--interval 2s] [--once] [--dry-run]
internal/paths/paths.go    SlackDir="slack", SlackConfig, SlackThreadmap
.gitignore                 .hatch/run/slack/
```

**Testability:** poster ẩn sau interface `poster{ post(from,threadTS,name,icon,text) (ts,error) }`. OUT (`mirrorOnce`), IN (`handleIncoming(incoming)`) và `translateMentions` là method thuần, test bằng fake poster + temp bus. Toàn bộ slack-go (multiPoster, socketmode, auth.test) nằm trong `runtime.go`.

## 8. Config & bảo mật

- **Hub app** (1 cái): scopes `chat:write`, `chat:write.customize`,
  `channels:history`, `connections:write`; bật Socket Mode; subscribe
  `message.channels`. Cho app-token `xapp-` (inbound) + bot-token `xoxb-`
  (escalation + fallback).
- **Mỗi agent một app**: scope `chat:write`, mời vào channel; bot-token `xoxb-`.
- Nạp qua env `HATCH_SLACK_{APP_TOKEN,BOT_TOKEN,CHANNEL,BOSS}` +
  `HATCH_SLACK_TOKEN_<AGENT>` (vd `HATCH_SLACK_TOKEN_CLAUDE_CODE`), hoặc
  `.hatch/run/slack/config.json` (field `agents{id→token}`).
- **Không hardcode, không commit token.** `.hatch/run/slack/` vào `.gitignore`.
- Đây là Slack workspace của chính boss, chính boss yêu cầu → không vướng rule
  "comm ra ngoài chưa review".

## 9. Giới hạn v1 (ngoài scope)

- `@codex`/`@claude`/`@agy`/`@kiro` từ Slack đều **auto-resume** (cả 4 kind đã có
  headless contract trong `daemon/runner.go`): claude `-p --resume`, codex
  `exec resume`, agy `-p --conversation`, kiro `kiro-cli chat --no-interactive
  --resume-id`. Chỉ `manual`/`user` là ghế tương tác, không bị daemon đánh thức.
  (kiro cần `KIRO_API_KEY` trong env. Contract cả agy lẫn kiro verify trực tiếp
  trên CLI đã cài: kiro `chat [OPTIONS] [INPUT]` + `--no-interactive` +
  `--resume-id <SESSION_ID>`.)
- OUT poll 2s (không fsnotify) — chấp nhận, đơn giản.
- Slack 1 cấp thread — `InReplyTo` bus không phản ánh vào Slack layout.
- Chạy `hatch slack` + `hatch daemon` là **hai lệnh**. `hatch up` gộp = sau.

## 10. DoD

- [ ] `make lint` xanh; `go test ./...` và `-tags hatch_legacy` đều xanh.
- [ ] Build sạch cả hai tag.
- [ ] `hatch slack --once` chạy OUT một nhịp không cần Socket Mode (smoke).
- [ ] Token không lọt vào git; `.hatch/run/slack/` gitignored.
- [ ] @tag reviewer khác, không tự merge (human gate).
