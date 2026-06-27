# Paperclip → Hatch: nghiên cứu & đề xuất chuyển sang Chat-first

_Ngày: 22/06/2026 · Tác giả: Conductor (claude-code) · Trạng thái: Proposal (chờ chốt)_
_Nguồn: đọc trực tiếp repo [paperclipai/paperclip](https://github.com/paperclipai/paperclip) (README, ROADMAP, `doc/TASKS.md`, `doc/TASKS-mcp.md`, `doc/execution-semantics.md`) + codebase Hatch hiện tại (`hatch/`, `.hatch/`)._

---

## 0. TL;DR

- **Paperclip không phải "task manager nhẹ".** Nó là một **control plane đầy đủ** để vận hành "AI company": Node server + React UI + Postgres, org chart, budget, governance, heartbeat scheduler. Khẩu hiệu của họ: _"If OpenClaw is an employee, Paperclip is the company."_
- **"Backlog-as-comms" của paperclip là hệ quả, không phải lựa chọn thẩm mỹ.** Vì paperclip **chủ động lái agent** (push: heartbeat scheduler wake agent dậy), nó **bắt buộc** phải có state máy-đọc-được: `status` enum, single-assignee = atomic checkout lock, blocker graph. Toàn bộ `execution-semantics.md` (700 dòng về checkout lock, stranded-work recovery, monitor, watchdog) tồn tại **chỉ vì** control plane chịu trách nhiệm liveness.
- **Hatch theo mô hình ngược lại: embedded, không điều khiển** (pull: agent đã chạy sẵn, tự lái, tự đọc bus). Khi bạn **không lái** agent, bạn **không cần** status/lock/heartbeat/recovery. Bạn cần một **cuộc hội thoại chung** mà các agent đang sống poll vào.
- **Vì vậy chat-first không chỉ là một UX choice — nó là kiến trúc đúng cho mô hình embedded.** Đề xuất: **chat = event log nguồn-sự-thật-duy-nhất; "task/board" = projection (reduce) trên chat, không phải store riêng.** Đây đúng là phép nghịch đảo của paperclip.
- **Tác động code: GIẢM, không tăng.** Hatch đang là hybrid (board lane store + ledger _và_ bus). Chat-first cho phép **xóa** board-lane-store + ledger, **nâng** các workflow verb (claim/done/block/review/handoff) từ prose trong CLAUDE.md thành **control message** hạng nhất, và biến `hatch board`/`status` thành **reducer** trên bus.

**Khuyến nghị: Option A — Chat-as-event-log + board-as-projection.** Chi tiết ở §7–§8.

---

## 1. Paperclip thực sự là gì

### 1.1 Định vị
Một control plane open-source để **điều phối một đội agent chạy cả một doanh nghiệp**. Bạn định nghĩa goal ("Build the #1 note app to $1M MRR"), "thuê" team (CEO/CTO/eng/designer — bất kỳ agent nào), set budget, bấm go, giám sát từ dashboard. _"Manage business goals, not pull requests."_

### 1.2 Kiến trúc (theo README "What's Under the Hood")
Một server đơn xử lý 12 subsystem: Identity & Access · Org Chart & Agents · Work & Tasks · **Heartbeat Execution** · Workspaces & Runtime · Governance & Approvals · Budget & Cost · Routines & Schedules · Plugins · Secrets & Storage · Activity & Events · Company Portability. Local: 1 Node process + embedded Postgres + file storage. Prod: Postgres riêng + deploy tùy ý.

### 1.3 Mô hình "backlog-as-communication"
Đây là phần cốt lõi cho câu hỏi của chúng ta. Đọc `doc/TASKS.md` + `doc/TASKS-mcp.md`:

- **Phân cấp entity:** `Initiative → Project → Milestone → Issue → Sub-issue`. Issue là đơn vị công việc lõi. Đây thực chất là một bản **clone Linear/Jira**.
- **Issue có state máy-đọc:** `status` (workflow state theo category triage/backlog/unstarted/started/completed/cancelled), `priority` (0–4 cố định), **single `assigneeId`** (chủ ý: "clear ownership prevents diffusion of responsibility"), `parentId`, `goalId`, blocker relations (`blocks`/`blocked_by`/`related`/`duplicate`).
- **Kênh giao tiếp giữa agent = artefact gắn vào issue:**
  - **Comments** (threaded, có `parentId`, `resolvedAt`) — "agents need to communicate about issues without overwriting the description".
  - **Assignment** — chuyển owner.
  - **@-mention / status transition / blocker** — tín hiệu điều phối.
- **MCP surface = 35 operations** xoay quanh CRUD issue/project/milestone/label/relation/comment/initiative. Agent "nói chuyện" bằng cách **mutate issue** và **comment trên issue**.

→ **"Đơn vị giao tiếp là cái issue."** Hội thoại là thứ _treo dưới_ một work item có cấu trúc cứng. Trạng thái phối hợp được biểu diễn **tường minh** bằng field (`status`, `assignee`, `blockedBy`).

### 1.4 Heartbeat & execution semantics — cái giá thật của backlog-first
`doc/execution-semantics.md` tách 4 khái niệm: **structure** (parent/sub), **dependency** (blocker), **ownership** (assignee), **execution** (control plane có đang có đường sống để đẩy issue đi không). Vì paperclip **dispatch run** cho agent (DB-backed wakeup queue, coalescing, budget check, workspace resolve, secret inject), nó phải gánh nguyên một bộ contract khổng lồ:

- **Atomic checkout:** `checkoutRunId` (lock quyền sở hữu) vs `executionRunId` (run đang sống) — compare-and-clear khi run terminal, không clear lock của run non-terminal, self-heal lock trỏ vào run đã chết.
- **Liveness contract:** issue agent-owned non-terminal **không bao giờ** được rơi vào trạng thái "không ai chịu trách nhiệm bước kế và không gì wake nó". Phải luôn trả lời được _"what moves this forward next?"_ qua một trong các primitive: active run / queued wake / typed participant / pending interaction / monitor / human owner / blocker chain / recovery action.
- **Crash recovery:** stranded `todo` (dispatch recovery, requeue 1 wake), stranded `in_progress` (continuation recovery), rồi mới escalate `blocked` + recovery action.
- **Watchdog:** task watchdog (subtree dừng), silent active-run watchdog (process còn sống nhưng im output), productivity review.

**Đây là toàn bộ độ phức tạp mà một _controller_ phải trả.** Nó hợp lý cho paperclip. Nó **vô nghĩa** với một harness embedded.

---

## 2. Insight cốt lõi: Push (controller) vs Pull (embedded)

| | **Paperclip** | **Hatch** |
|---|---|---|
| Quan hệ với agent | **Controller** — spawn & wake agent qua heartbeat | **Embedded** — agent đã chạy sẵn (terminal của user), tự lái |
| Mô hình | **Push:** scheduler đẩy việc vào agent | **Pull:** agent đọc bus rồi tự hành động |
| Vì sao cần state máy-đọc | Để scheduler **lock, schedule, budget, recover** | Không có scheduler nào tiêu thụ state → **không cần** |
| Liveness do ai lo | Control plane (execution-semantics.md) | **Con người + agent conductor** đọc inbox |
| Nếu agent chết | Recovery contract tự requeue/escalate | User thấy ngay (đó là terminal của họ) |
| Hệ quả comms tự nhiên | **Backlog-first** (issue là object cứng) | **Chat-first** (message là object) |

> **Điểm chốt cho CTO:** chat-vs-backlog **không phải** là quyết định gốc. Quyết định gốc là **"có lái agent hay không"** — và Hatch đã chốt "không điều khiển" ngay trong charter L0. Chat-first chỉ là hệ quả tất yếu. Chọn backlog-first cho một harness không-lái-agent = trả giá schema + liveness/recovery contract mà **chẳng có ai tiêu thụ** state đó. Đó là complexity thuần lỗ.

---

## 3. Mô hình Chat-first đề xuất cho Hatch

### 3.1 Nguyên lý: Chat là event log, task/board là projection
- **Bus (chat) = append-only event log, source-of-truth duy nhất.** Mỗi thread = một task. Mỗi message = một event bất biến.
- **"Task state" không được lưu — nó được _tính_** bằng cách reduce/fold các message trong thread (event sourcing). `hatch board` = chạy reducer trên tất cả thread, suy ra lane của từng thread từ **control message mới nhất**.
- **Ledger biến mất** vì bus _chính là_ ledger (append-only + auditable + có `why` trong body).

Điều này nghịch đảo paperclip: ở paperclip chat treo dưới issue; ở đây **state treo dưới chat**.

### 3.2 Nâng workflow verb từ prose → control message
Hiện `model.Message.Type` chỉ có `msg/ask/reply/decision`. Các verb scrum (claim/handoff/done/review/block/unblock) đang **nằm trong CLAUDE.md dưới dạng văn xuôi** — không máy nào đọc được, nên `board` không thể suy ra trạng thái thật. Đề xuất thêm một nhóm **control message** (structured verb) bên cạnh message hội thoại:

```
Hội thoại (đã có): msg · ask · reply · decision
Control   (thêm) : claim · handoff · done · block · unblock · review
```

Một control message vẫn là một message bình thường (cùng thread, cùng `From`, có `Body` giải thích `why`), chỉ khác là reducer hiểu nó như một **state transition**. Ví dụ reduce một thread:

```
open(T-3, "Fix slugify")              → lane=backlog, owner=∅
claim   by @codex                     → lane=in-progress, owner=codex
block   re:T-1 "chờ API contract"     → lane=blocked, blockedBy=[T-1]
unblock (T-1 done)                    → lane=in-progress
handoff to @reviewer "tests xanh"     → lane=review, owner=reviewer
done    by @reviewer "DoD met"        → lane=done
```

`board`/`status` = `reduce(thread)` cho mọi thread. Không có lane store. Không thể lệch sync (chỉ có một nguồn).

### 3.3 Bảng chiếu: paperclip primitive → Hatch chat-first

| Bài toán | Paperclip (backlog-first) | Hatch (chat-first) |
|---|---|---|
| Đơn vị công việc | Issue (entity cứng) | Thread (chuỗi message) |
| Trạng thái | `status` field (lưu) | Reduce control message (tính) |
| Ai sở hữu | `assigneeId` (1 dòng SQL) | `owner` = người `claim` gần nhất (tính) |
| Phụ thuộc | blocker relation table | `block re:T-x` message → wait graph (tính) |
| Phân rã việc | sub-issue (`parentId`) | thread con tham chiếu `parent:T-x` |
| Giao tiếp | comment trên issue | message trong thread (vốn là cốt lõi) |
| "Việc gì kế tiếp?" | liveness contract + heartbeat | board projection đánh dấu thread "stalled" (no owner / no activity) cho **người** xử lý |
| Audit | activity/events table | bus append-only (sẵn có) |

---

## 4. Những bài toán chat-first PHẢI tự giải (paperclip được schema cho không)

Trung thực: backlog-first cho không 3 thứ mà chat-first phải chủ động giải. Đây là "ai trả giá".

1. **Truy vấn trạng thái nhanh.**
   - Mất gì: không `SELECT status`. Phải reduce thread mỗi lần hỏi "T-3 đang sao?".
   - Giải lean: reducer rẻ (thread thường ngắn); cache projection vào `.hatch/bus/.board.json` (derived cache, xóa được, không phải SoT) nếu cần tốc độ. **Không** coi cache là nguồn.

2. **Chống double-work / ownership race.**
   - Mất gì: không có atomic checkout. Hai agent có thể `claim` cùng thread.
   - Giải lean: **advisory claim** — reducer coi "claim hợp lệ = claim đầu tiên sau lần open/handoff gần nhất"; claim sau thấy đã có owner thì lùi (đúng tinh thần `409` của paperclip nhưng **bằng convention + projection**, không bằng engine cưỡng chế). Đủ tốt cho squad nhỏ; race là sự kiện hiếm và **nhìn thấy được** trên board.

3. **Liveness "việc gì kế tiếp?"**
   - Mất gì: không heartbeat/recovery.
   - Giải lean: **không cần auto-recovery** (embedded — agent là terminal sống, user thấy khi nó chết). Thay vào đó board projection **đánh dấu** thread `in-progress` mà không có owner hoặc im lặng quá ngưỡng là **"stalled"** — để **người/conductor** xử lý. Mượn _câu hỏi_ "what moves this forward next?" của paperclip làm **design check cho view**, không phải làm engine.

4. **Kỷ luật posting.**
   - Mất gì: paperclip _ép_ state; Hatch _thuyết phục_ (protocol prose).
   - Giải lean: làm control verb thành **MCP tool một-phát rẻ** (`chat_claim`, `chat_done`, `chat_block`…) để post state có cấu trúc chỉ tốn 1 call; board làm chỗ thiếu sót **lộ ra** ("T-3: no owner, stale 2h"). Bù enforcement bằng visibility.

---

## 5. Lấy gì / Bỏ gì từ paperclip

**LẤY (kể cả khi chat-first):**
- **Human-readable id** kiểu `T-3` thay cho `m0622-150405-xxxxxx` hiện tại — cực hợp với chat ("xử T-3 đi" > đọc UUID).
- **Câu hỏi liveness** _"what moves this forward next?"_ làm tiêu chí thiết kế cho board view (stalled detection), **tính chứ không ép**.
- **Single-owner clarity** (dù mềm): board luôn hiện owner hiện tại của thread.
- **Blocker là quan hệ hạng nhất** (dù derived từ `block re:T-x`).
- **Context/goal chảy xuống**: charter L0 + `parent:T-x` cho thread con (đã có context map).

**BỎ (đây là cái giá controller, embedded không trả):**
- Heartbeat / run / checkout-lock / stranded-work recovery / watchdog (toàn bộ `execution-semantics.md`).
- Postgres + Node server + React dashboard → giữ Go single-binary + filesystem.
- Org chart / budget / governance **as engine** → Hatch làm gate bằng self-check prose (DoD), giữ nguyên.
- Phân cấp 5 tầng Initiative→Project→Milestone→Issue→Sub-issue → thừa cho squad 1-repo. Thread + `parent:` là đủ.

---

## 6. Tác động cụ thể lên codebase Hatch

| Khu vực | Hiện tại | Chat-first |
|---|---|---|
| `internal/bus` | Append-only thread store (đã tốt) | **Giữ, là trung tâm.** Thêm control verb. |
| `internal/model/message.go` | type: msg/ask/reply/decision | **Thêm:** claim/handoff/done/block/unblock/review + (tùy) field `Ref`, `Parent`. |
| `internal/store` board lane (`board/{backlog,…}`) | Store riêng | **Xóa** → thay bằng reducer `wf.Board = reduce(bus)`. |
| `internal/model/ledger.go` + ledger store | Audit riêng | **Xóa** → bus _là_ ledger. |
| `internal/wf` engine (Move/Escalate) | Đổi lane trên store | **Đổi** thành "post control message" + projection. |
| `hatch board` / `status` / TUI | Đọc lane store | **Đọc reduce(bus)**. |
| MCP server | chat_*/kb_* | **Thêm** chat_claim/done/block… (thin wrapper post control msg). |
| Thread id (`newID`) | `m0622-…` | **Đổi** sang `T-{seq}` human-readable. |

→ Ròng là **xóa hai store** (board lane + ledger), **thêm vài message kind + reducer**. Khớp charter: _"Minimum code, surgical"_ và Lean Hexagonal (seam `store behind wf.Board` đúng chỗ để thay adapter).

---

## 7. Ba lựa chọn & trade-off (ai trả giá)

### Option A — Chat-as-event-log + board-as-projection ✅ **(khuyến nghị)**
- **Là gì:** bus là SoT duy nhất; board/ledger thành projection; workflow verb thành control message; advisory claim; stalled detection tính trên view.
- **Được:** đúng mô hình embedded; **giảm code** (xóa 2 store); audit miễn phí; không bao giờ lệch sync; con người đọc native; không schema migration.
- **Ai trả giá:** mất truy vấn state O(1) (phải reduce — rẻ, có thể cache); ownership là advisory (race hiếm, nhìn thấy được). _Người trả: agent phải có kỷ luật post control msg — bù bằng MCP tool rẻ + board phơi thiếu sót._

### Option B — Hybrid: giữ cả board store lẫn chat, chat làm "primary", board làm cache "đồng bộ"
- **Là gì:** đúng trạng thái Hatch hiện tại, chỉ tuyên bố chat ưu tiên.
- **Được:** truy vấn nhanh từ board cache.
- **Ai trả giá:** **hai nguồn sự thật phải sync** = ổ bug kinh điển; nhiều code hơn; "ai đúng khi lệch?" không có câu trả lời sạch. **Tệ nhất cả đôi.** Không khuyến nghị.

### Option C — Backlog-first như paperclip (issue là object, chat treo dưới)
- **Là gì:** clone mô hình paperclip ở quy mô nhỏ.
- **Được:** state máy-đọc, ownership cứng, thân thiện scheduler.
- **Ai trả giá:** phải xây + nuôi schema + **toàn bộ liveness/recovery contract** — mà Hatch **không có scheduler nào tiêu thụ**. Complexity thuần lỗ, đi ngược charter "embedded, không điều khiển". Không khuyến nghị.

---

## 8. Rủi ro & giảm thiểu (Option A)

| Rủi ro | Giảm thiểu |
|---|---|
| Agent quên post control message → board sai | MCP tool 1-call (`chat_done`…); board đánh dấu thread `in-progress` im lặng là "stalled" để lộ thiếu sót |
| Ownership race (2 agent cùng claim) | Reducer: claim hợp lệ = claim đầu sau open/handoff; claim sau tự lùi; race hiếm & hiển thị |
| Reduce chậm khi nhiều/đầy thread | Thread thường ngắn; thêm derived cache `.board.json` (xóa được) nếu đo thấy chậm — **không** coi là SoT |
| Mất tính năng query phức tạp (filter theo label/priority…) | Thêm dần như field trong control msg khi thực sự cần; **không** xây trước (charter: nothing speculative) |
| Migration board/ledger cũ | Viết một lần: replay lane hiện tại thành control message vào thread tương ứng |

---

## 9. Next steps đề xuất (nếu chốt Option A)

1. **Chốt decision** → ghi ADR (`kb_add type=decision` hoặc `kb/decisions/`).
2. **Spec reducer:** định nghĩa chính xác fold (control msg → lane/owner/blockedBy) — đây là "engine" mới, thuần hàm, dễ test.
3. **Mở rộng `model.Message`** với control kind + `Ref`/`Parent`; thêm `T-{seq}` id.
4. **Viết `wf.Board = reduce(bus)`** thay store lane; xóa board-lane store + ledger.
5. **Thêm MCP tool** control verb (thin wrapper).
6. **Cập nhật SSOT** (charter/workflow) + `hatch compile` để protocol prose khớp mô hình mới.
7. **Migration** lane/ledger cũ → control message (chạy 1 lần).

---

## 10. Một dòng để nhớ

> Paperclip đặt **chat dưới task** vì nó **lái** agent. Hatch nên đặt **task dưới chat** vì nó **không lái** — task chỉ là cái bóng (projection) mà cuộc hội thoại đổ xuống. Chọn đúng nghịch đảo này là chọn đúng độ phức tạp mình phải nuôi.
