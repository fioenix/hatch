# Hatch Session Store — design spec

_Status: approved (model + capture-scope confirmed 2026-06-22)_

## 1. Vấn đề

Một Slack channel = một project = một `.hatch` workspace; trong đó mỗi agent có
**rất nhiều session** theo thời gian. `roster.json` cũ chỉ giữ **một** `SessionID`
mutable/agent — không history, không validate sống/chết, không biểu diễn được
nhiều session. Daemon resume mù id đó; chết thì wake fail.

## 2. Quyết định

| Nhánh | Chọn |
|---|---|
| Đơn vị session | **per (agent, bus-thread)** — `S(codex, t-42)`. Channel = project, mỗi task-thread một "trí nhớ". |
| Triết lý | **session = warm cache; KB/bus = SoT.** Mất session ⇒ đọc lại record, không mất việc. |
| Capture scope v1 | **claude+codex warm; agy+kiro stateless.** |
| Parallelism (wip>1) | **hoãn.** Giữ daemon serialize per-member; mỗi wake resume đúng session theo thread. |

## 3. Ma trận năng lực CLI (verified trên binary đã cài)

| Kind | Lấy id | Resume | Chiến lược |
|---|---|---|---|
| claude | `--session-id <uuid>` (assign) | `--resume <uuid>` | **assign** |
| codex | `exec --json` → `session_meta.payload.id` | `exec resume <id>` | **capture** |
| kiro | `--list-sessions -f json` (newest, racy) | `--resume-id <id>` | stateless (capture hoãn) |
| agy | không lộ id | `--conversation <id>` (nếu biết) | **stateless** |

## 4. Cấu phần & file

```
.hatch/sessions.json            { agent → { thread → Session } }   (gitignored, runtime)
internal/model/session.go       Session{Agent,Thread,Kind,ID,Status,StartedAt,LastResumedAt,History}
internal/session/store.go       Store: Get/Put(+history)/MarkStale/All; atomic write. + store_test.go
internal/daemon/runner.go       planWake(m,thread,prior,prompt) → wakePlan{argv,headless,capture,assignID};
                                ExecRunner{…,Sessions}: plan→run→capture→commit; sessionCapture writer; uuid4
internal/cli/sessionscmd.go     `hatch sessions` view (tabwriter)
internal/cli/daemoncmd.go       inject session.New(layout) vào ExecRunner
```

## 5. Lifecycle (trong ExecRunner.Wake)

1. `thread := threadOf(payload)` = channel của tin mới nhất trong payload.
2. `prior := store.Get(agent, thread)`.
3. `planWake`: prior **live** ⇒ resume warm; ngược lại fresh (assign claude / capture codex / stateless agy,kiro).
4. Chạy CLI. Nếu capture: `sessionCapture` vừa forward stdout vừa rút `session_meta.id` (best-effort).
5. `commit`:
   - assign ok ⇒ Put live{assignID}.
   - capture ok & bắt được id ⇒ Put live{id}.
   - resume ok ⇒ bump `LastResumedAt`.
   - resume lỗi ⇒ `MarkStale` ⇒ wake sau bắt đầu fresh (self-heal across wakes).
   - stateless ⇒ không lưu gì.

## 6. Bất biến

- **Best-effort capture**: format drift ⇒ không lưu ⇒ wake sau fresh, **không bao giờ crash**.
- **Stale self-heal**: resume chết ⇒ stale ⇒ fresh ở wake kế (không re-run giữa wake).
- **KB/bus là SoT**: stateless agent (agy) đọc lại record mỗi wake vẫn đúng.
- **History**: Put đè id khác ⇒ đẩy id cũ vào `History` (audit "nhiều session").

## 7. Hoãn (ngoài v1)

- **Parallelism per-thread (wip>1)**: cần đổi working-set của daemon + wake-policy
  sang key `(member, thread)`; `registry.wip` khi đó = trần session đồng thời/agent.
  V1 vẫn serialize per-member (một agent xử lý tuần tự từng thread).
- **kiro capture** qua `--list-sessions` (heuristic newest có race).
- **Liveness proactive**: hiện chỉ phát hiện chết khi resume lỗi.

## 8. DoD

- [x] `make lint` xanh; test default + `hatch_legacy` đều xanh; build cả hai tag.
- [x] store round-trip/history/stale/All; planWake assign/capture/resume/stateless;
  sessionCapture parse `session_meta` qua 2 chunk.
- [x] `hatch sessions` chạy (rỗng → hint). `.hatch/sessions.json` gitignored.
- [ ] @tag reviewer khác; không tự merge (human gate).
