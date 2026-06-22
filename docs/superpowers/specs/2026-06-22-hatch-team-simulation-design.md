# Hatch — Team Simulation Design (chat-first, peer-to-peer, wake-on-message)

_Ngày: 22/06/2026 · Trạng thái: Approved design → implementation_
_Tiền đề: [nghiên cứu paperclip vs overclaud](../../research/2026-06-22-paperclip-vs-overclaud-chat-first.md). Đã brainstorm + chốt với owner qua nhiều vòng._

---

## 1. North star

**Hatch = một văn phòng remote async cho một squad coding-agent CLI làm chung một repo.** Không phải hệ-thống-xoay-quanh-task (paperclip). Khi user `hatch init` một project = mở một **căn phòng chung**: các agent CLI khác nhau (Claude Code, Codex, agy, Kiro) **join**, **nhận biết sự tồn tại của nhau** (presence/roster), **hiểu cùng một context** (repo + chat + docs/KB). Khi cần phối hợp, chúng **chat trực tiếp với nhau** — không qua engine điều phối. Triết lý: **mô phỏng một team người làm việc thật**, với **ownership cao** và **kỷ luật chuyên nghiệp**, không chat cho vui, không đùn đẩy.

## 2. Nguyên tắc cốt lõi (SSOT-level)

1. **Tách bạch hai thứ hay bị lẫn:**
   - **Work-orchestration** (quyết định *ai-làm-gì*: assign, lane-engine, checkout-lock): **KHÔNG có.** Không engine nào tự sinh/giao việc.
   - **Message-delivery + wake** (đưa tin tới đúng người, cấp cho họ một lượt để đáp): **CÓ.** Đây là vai "chat server" — mọi team chat đều cần. Wake **luôn là hệ quả của một message ai đó chủ động gửi**, không bao giờ do scheduler tự phát.
2. **Không có sếp-phần-mềm, nhưng luôn có sếp-người = user.** User là boss: đặt mục tiêu, ưu tiên, duyệt, bẻ lái. Phần mềm không thay quyền đó.
3. **Chat là kênh tương tác chính. Task-management bị giáng vai** xuống lớp plan/docs/note (artifact để *lập kế hoạch & ghi nhớ*), không phải kênh điều phối. Không còn lane backlog→in-progress→review→done như một engine.
4. **Embedded, lean, surgical.** Go single-binary, filesystem-as-DB. Lean Hexagonal (model/port/adapter). Wake daemon là thành phần MỚI trong **build mặc định**, không revive `orchestrator` cũ (đang sau `hatch_legacy`).
5. **Norm + Backstop.** Hành vi chuyên nghiệp được dạy bằng **norm** (prose compile xuống mọi agent) và được giữ bằng **backstop** (cơ chế trong daemon/MCP). Prose-suông không đủ để đảm bảo ownership.

## 3. Kiến trúc tổng thể

```
┌──────────────────────────────────────────────────────────────────────┐
│  WORKSPACE  (1 repo · 1 .hatch)                  "căn phòng của team"  │
│   THÀNH VIÊN: claude · codex · agy · kiro · USER  (mỗi agent = session │
│               có trí nhớ riêng; idle thì ngủ, bị gọi thì thức)         │
│        └──────────── đọc/ghi qua MCP ───────────┐                      │
│        ┌────────────────────────────────────────▼────────┐            │
│        │     HATCH MCP SERVER  +  WAKE DAEMON             │            │
│        │  whoami·join·roster·chat_*·kb_*  |  wake policy  │            │
│        └──┬──────────────┬───────────────┬───────┬────────┘            │
│   ┌───────▼─┐  ┌─────────▼──┐  ┌──────────▼─┐  ┌──▼──────────┐         │
│   │ PRESENCE│  │  CHAT BUS  │  │  DOCS / KB │  │  GIT REPO   │         │
│   │ /ROSTER │  │  ⭐ chính  │  │ plan·spec· │  │  code chung │         │
│   │ ai ở đây│  │ #chan·DM·  │  │ note·ADR = │  │             │         │
│   │ on/idle │  │ thread·@   │  │ task layer │  │             │         │
│   └─────────┘  └────────────┘  └────────────┘  └─────────────┘         │
└──────────────────────────────────────────────────────────────────────┘
```

Năm thành phần:

| # | Thành phần | Vai | Trạng thái code |
|---|---|---|---|
| 1 | **Workspace** | Căn phòng: repo + `.hatch` (chat + docs/KB + roster) | Có (`hatch init`) |
| 2 | **Presence / Roster** | Ai đã join, on/idle, vai trò, last-seen | **MỚI trong build mặc định** (legacy có bản khác) |
| 3 | **Chat bus** ⭐ | Kênh tương tác chính: channel/DM/thread/@mention/inbox | Có (`internal/bus`) — giữ làm trung tâm |
| 4 | **Wake daemon** | Theo bus, @mention → wake đúng agent một lượt. Chỉ delivery | **MỚI** — keystone |
| 5 | **Docs / KB** | Task giáng vai: plan/spec/note/decision | Có (`kb/` + `templates/docs/`) — giữ; gỡ lane-engine |

## 4. Vòng đời teammate (state machine)

**Mặc định: resume-on-wake** ("ngủ khi rảnh, thức khi bị gọi, giữ trí nhớ").

```
                 bị @mention / DM / reply-vào-ask-đang-mở
                                  │
   ┌──────────────┐  wake: resume  ┌──────────────┐  post xong → ngủ
   │  SUSPENDED   │  session + đẩy  │   WORKING    │
   │  ngủ · $0    ├───────────────►│ một lượt:    ├──────┐
   │  trí nhớ lưu │                 │ đọc phòng →  │      │
   │ trong session│◄───────────────┤ hành động →  │◄─────┘
   └──────────────┘                 │ post reply   │
                                     └──────────────┘
```

- **Continuity có thật** qua resume: `claude --resume <id> -p`, `codex resume`/`codex exec`.
- **Idle = $0** (session suspended; không đốt token).
- **Optional "live mode"** (Claude Code `--channels`): giữ session ấm, push tin vào để đáp gần-tức-thì. Dùng cho ghế user đang pair. **Không** phải mặc định.
- **User = thành viên người luôn-online**; agent user đang gõ cùng chỉ là "người đang pair", về kiến trúc mọi agent đều wake-on-message.

### Mapping kỹ thuật (đã verify)

| Khả năng | Claude Code | Codex |
|---|---|---|
| Headless 1 lượt | `claude -p "…"` (`--output-format json/stream-json`) | `codex exec "…"` |
| Resume giữ trí nhớ | `claude --resume <id>` / `-c` | `codex resume --last` / `resume <id>` |
| Wake từ ngoài | `--channels` (MCP `claude/channel` push vào session đang chạy) | `app-server`/`remote-control` (experimental); hoặc resume-exec |
| Đọc/ghi chat | MCP stdio client (`.mcp.json`) | `codex mcp` (MCP client) |
| Session store | `~/.claude/projects/<proj>/sessions/` | session id từ `codex` |

→ Mặc định dùng **resume-exec** (tương thích cả hai). `--channels` là nâng cấp live-mode chỉ cho Claude Code.

## 5. Wake Policy (cơ chế delivery — backstop chống wake-storm)

Đặt trong **domain** dưới dạng **hàm thuần, test được**; daemon chỉ là vòng lặp gọi nó.

```
Input:  roster (ai on/idle/role), tập message mới chưa giao, trạng thái wake
        hiện hành (ai đang working, cursor mỗi agent, cascade depth, rate window)
Output: danh sách WakeDecision{agent, reason, payload[], holdUntil?}  +  escalations

LUẬT:
 1. TRIGGER hợp lệ để wake một agent: chỉ khi message
      (a) @mention đích danh agent đó, hoặc
      (b) reply vào một ask đang-mở của agent đó, hoặc
      (c) DM tới agent đó.
    Message không @ai và không phải (b)/(c) → KHÔNG wake ai (chỉ lưu vào bus).
 2. COALESCE: nhiều message tới cùng một agent đang SUSPENDED trong cửa sổ W
    → gộp thành MỘT wake mang tất cả payload.
 3. DEBOUNCE: agent đang WORKING → không wake; xếp payload vào hàng đợi,
    giao ở ranh giới lượt kế tiếp của agent đó.
 4. NO SELF / ECHO WAKE: post của chính mình không wake mình; reply không
    re-wake người vừa nói trừ khi bị @ đích danh.
 5. DEPTH LIMIT: mỗi "episode" (cây hội thoại bắt nguồn từ 1 tin của con người
    hoặc 1 thread root) mang cascade depth; auto-wake vượt D → HOLD + escalate user.
 6. RATE CAP: trần wake/phút mỗi agent; vượt → xếp hàng (HOLD), không drop.
 7. LOOP BREAK: một cặp agent ping-pong ≥ N vòng trong một episode mà không có
    deliverable / decision-message / thay-đổi-repo → cắt auto-wake, escalate user.
```

Tham số mặc định (config được): `W=2s`, `D=4`, `RateCap=6/phút`, `N=3 vòng`.

## 6. Working Agreement (norm — định nghĩa "chuyên nghiệp")

Nâng thành **mục SSOT mới** `.hatch/working-agreement.md`, compile xuống mọi agent (như charter/roles). Sáu điều, mỗi điều ánh xạ tới backstop:

| # | Norm | Backstop |
|---|---|---|
| 1 | **Bias to action** — việc reversible thì làm, hỏi chỉ khi irreversible/mơ hồ tốn kém | — (văn hoá) |
| 2 | **Mỗi tin xứng token của nó** — info/quyết định/deliverable/câu hỏi cụ thể; cấm "ok/thanks/noted" | Wake Policy #1 (tin rỗng không wake ai) |
| 3 | **Own cái mày mở / cái mày bị nhờ** — handoff phải *có tên người kế* + *được ack* | Roster "current owner" mỗi thread; view phơi thread mồ côi |
| 4 | **Close the loop** — xong thì post kết quả; kẹt thì post blocker + cần ai; không im lặng | Daemon nudge owner 1 lần khi thread đứng, rồi escalate user |
| 5 | **Self-check DoD trước khi báo xong** — tag reviewer khác, không tự duyệt | (đã có trong CLAUDE.md DoD) |
| 6 | **Escalate sớm lên user** khi kẹt ở một *quyết định* | Wake Policy #5/#7 (depth/loop → user) |

## 7. Governance / human-in-the-loop (2 giai đoạn)

Cùng một bộ máy (bus + daemon + presence); chỉ khác **điểm vào** của con người.

- **GĐ1 — Boss qua proxy (MVP, không cần UI mới).** User nói chuyện với một agent đóng vai **team-leader** (mặc định Claude Code = role `conductor`). Team-leader nhận việc → có thể **viết plan doc** → điều phối **qua chat** (post `@codex @agy` như peer). Team-leader là **delegate của boss**, KHÔNG phải engine.
- **GĐ2 — Boss vào thẳng phòng (roadmap).** Một **chat UI kiểu Slack** render bus; user là member hạng nhất, thấy mọi chat của agent, `@tag` thẳng bất kỳ agent để bơm prompt. Daemon giao `@codex` của user y như của một peer.

**Bất biến:** GĐ2 = GĐ1 với con người dời từ "ngoài, qua proxy" vào "trong phòng, trực tiếp". Lõi không đổi. Role `conductor` KHÔNG bị xóa — **reframe** thành delegate của boss (khớp `claude-code = Conductor mặc định` đang có).

## 8. Thay đổi SSOT (charter & roles)

- **`charter.md`:** thêm nguyên tắc tách *work-orchestration* (không) vs *delivery/wake* (có); ghi "không sếp-phần-mềm, có sếp-người = user"; nêu chat là kênh chính, task = plan/docs layer.
- **`workflow.yaml`:** bỏ mô hình lane-engine như cơ chế; thay bằng mô tả "phối hợp qua chat + Working Agreement + Wake Policy". Giữ phần nghi thức (planning/retro/standup) như *thói quen*, không phải engine.
- **`working-agreement.md` (mới):** 6 điều ở §6.
- **`roles/conductor.md`:** reframe = delegate của boss (proxy GĐ1), điều phối qua chat, không tự merge/không engine.
- Sau khi sửa: `hatch compile` đẩy xuống CLAUDE.md/AGENTS.md/…

## 9. Build / Keep / Remove (cụ thể theo package)

Tất cả phần mới nhắm **build mặc định** (không tag). Phải build sạch cả `default` lẫn `-tags hatch_legacy`.

| Hành động | Chi tiết |
|---|---|
| **KEEP** | `internal/bus` (trung tâm), `internal/mcpserver` (mở rộng), `internal/store` (KB/docs), `internal/compile`, `internal/model.Message`, `internal/tui` (chat view) |
| **ADD (default)** | `internal/model`: `Member`/`Roster` (presence-in-room), `WakeDecision`; `internal/wake` (Wake Policy thuần hàm + decider); `internal/roster` (store roster filesystem); daemon loop trong `cmd/hatch` (`hatch watch`); MCP tools `join`/`roster`/`leave` |
| **ADAPTER** | wake spawning dùng *khái niệm* `Adapter.Build(RunRequest)→Invocation` nhưng tách bản default-build gọn (không kéo `orchestrator.Run` legacy dính ticket/ledger). Một `wake.Runner` port; adapter per kind (claude/codex/agy/mock) build lệnh resume-exec |
| **REMOVE / KHÔNG dùng ở default** | lane-engine `wf` (đã ở legacy) — không revive; ledger như SoT điều phối — không revive; board-lane CLI |
| **SSOT** | sửa charter/workflow/roles + thêm working-agreement; compile |

## 10. Data model (domain mới)

```go
// internal/model/member.go  (build mặc định)
type Member struct {
    ID        string   // "codex"
    Kind      string   // "claude" | "codex" | "agy" | "kiro" | "mock"
    Roles     []string // ["implementer","reviewer"]
    SessionID string   // id session resumable (trí nhớ); "" nếu chưa có
    Status    string   // online | idle | suspended | offline
    LastSeen  string   // RFC3339
    Note      string
}
type Roster map[string]Member

// internal/model/wake.go
type WakeReason string // "mention" | "reply_to_open_ask" | "dm" | "nudge"
type WakeDecision struct {
    Agent     string
    Reason    WakeReason
    Payload   []Message // message gây wake (đã coalesce)
    HoldUntil string    // nếu bị rate/depth hold
}
type Escalation struct {
    Episode string
    Cause   string // "depth_limit" | "loop_break" | "stalled_owner"
    To      string // thường là user/boss
    Note    string
}
```

`internal/wake` (thuần hàm):
```go
type State struct {
    Working   map[string]bool   // agent đang working
    Cursors   map[string]string // agent → ts đã giao tới
    Depth     map[string]int    // episode → cascade depth
    Rate:     map[string][]time.Time
    PingPong  map[string]int    // "a|b|episode" → số vòng
}
type Config struct{ W time.Duration; Depth, RateCap, LoopRounds int }
func Decide(r model.Roster, newMsgs []model.Message, st State, cfg Config)
    (decisions []model.WakeDecision, esc []model.Escalation, next State)
```
Decider không có IO — test bằng bảng input/output. Daemon là adapter quanh nó.

## 11. MCP surface (mở rộng)

Giữ: `whoami · chat_open · chat_post · chat_read · chat_inbox · chat_search · chat_channels · kb_add · kb_search`.
Thêm:
- `join {kind, roles[], session_id?}` → đăng ký Member vào roster, set `online`, lưu `session_id` để wake resume.
- `roster {}` → liệt kê thành viên + status + roles + last_seen (để agent "biết ai ở trong phòng").
- `leave {}` → set `offline`.
- (heartbeat ngầm) mọi tool call cập nhật `LastSeen`; idle quá ngưỡng → `idle`.

## 12. Luồng dữ liệu — một episode phối hợp

```
USER →(GĐ1) team-leader: "review auth.go giúp"
 team-leader: viết plan note (docs) nếu cần; post thread T "@codex review auth.go"
 daemon: thấy @codex (Wake Policy #1) → codex SUSPENDED → COALESCE → resume-exec codex
 codex: join/whoami → roster → đọc thread T + repo (MCP) → review → post reply vào T
 daemon: reply có @team-leader? nếu có → wake; nếu chỉ là kết quả → không cascade (#4)
 team-leader: tổng hợp, báo user. Episode đóng khi có deliverable/decision.
```

## 13. Implementation plan (phân pha, surgical)

**Phase A — Roster (foundation, build mặc định).**
- `model.Member`/`Roster`; `internal/roster` store (`.hatch/roster.json`); MCP `join`/`roster`/`leave`; `hatch roster` view; cập nhật `LastSeen` qua MCP. Unit + integration test. *Shippable độc lập.*

**Phase B — Wake Policy (domain thuần).**
- `model.WakeDecision`/`Escalation`; `internal/wake.Decide` + `State`/`Config`. Bảng test đầy đủ 7 luật. *Không IO, không cần daemon.*

**Phase C — Wake daemon (adapter + loop).**
- `wake.Runner` port + adapter per kind (resume-exec build lệnh; `mock` cho test). `hatch watch`: tail bus → `wake.Decide` → spawn qua Runner → mark cursor. Escalation → post vào DM của user. Tích hợp test với `mock` runner.

**Phase D — SSOT + compile.**
- Sửa charter/workflow/roles + thêm working-agreement; `hatch compile`; cập nhật CLAUDE.md output. Snapshot test compile.

**Phase E (roadmap, ngoài MVP) — live-mode `--channels` + Slack-like UI (GĐ2).**

## 14. Testing

- **wake.Decide:** bảng input→output cho cả 7 luật (mention/echo/coalesce/debounce/depth/rate/loop). Đây là phần logic dày nhất, phải phủ kỹ.
- **roster:** join/leave/idle-timeout/last-seen.
- **daemon:** dùng `mock` runner (không spawn thật) + bus tạm; xác minh wake đúng agent, coalesce, escalate.
- **MCP:** test `join`/`roster` qua server_test hiện có.
- **DoD:** `make lint && make test` xanh cả default lẫn `-tags hatch_legacy`; `go build ./...` cả hai tag.

## 15. Rủi ro & quyết định mở

- **Spawn lại trong default build** mâu thuẫn "embedded, không điều khiển" cũ → giải bằng tách work-orchestration vs delivery (§2.1) và sửa charter (§8). Đây là quyết định kiến trúc có chủ đích, không phải revert.
- **Resume-exec mỗi wake = cold-start cost** (khởi động CLI mỗi lượt). Chấp nhận ở MVP; live-mode `--channels` (Phase E) là tối ưu sau.
- **agy/kiro headless contract chưa chắc** → adapter trả `manual` handoff khi không có lệnh headless; chỉ claude/codex chạy auto ở MVP.
- **Episode boundary** (để tính depth/loop) định nghĩa = cây reply bắt nguồn từ một tin của người, hoặc thread root. Cần chốt khi code Phase B.
- **Wake một session đã tắt** chỉ làm được qua resume-exec (spawn mới giữ trí nhớ), không qua `--channels`. Mặc định resume-exec nên không vướng.

---

## Phụ lục — bất biến để không lạc hướng khi code

1. Daemon **chỉ** wake do một message chủ động; **không** tự sinh việc.
2. **Không** assign/lane/lock. "Owner" là *suy ra* từ chat (ai claim/open gần nhất), chỉ để hiển thị + nudge.
3. Phần mới ở **build mặc định**; build sạch cả hai tag.
4. Logic wake là **hàm thuần** (`internal/wake`); spawning là **adapter**.
5. Mọi norm sống trong **SSOT → compile**, không hard-code vào engine.
