# 18 — Quan sát agents đang làm việc (observability)

> **Trạng thái: ĐÃ IMPLEMENT** — transcript mỗi run + `hatch logs -f`, TUI tầng A (`hatch board`: board+live+activity), và mux tầng B (`hatch run --mux=tmux|zellij`) đều chạy. Doc này là thiết kế gốc.

## Hai thứ KHÁC NHAU cần quan sát

1. **Squad state** — board, ledger, bus, ai đang giữ ticket nào, cost/workload. Đây là *trạng thái đội*, đọc từ artifact. → Hatch tự render được (đã có `hatch board` TUI; thêm pane ledger/bus feed).
2. **Live output từng agent** — luồng stdout/stderr *trực tiếp* của process agent (claude/codex/…) trong lúc nó chạy. Đây là *quá trình*, là stream.

Hai cái này cần cơ chế khác nhau → có **hai tầng quan sát**, không loại trừ nhau.

## Tầng A — TUI một-process (mặc định, không phụ thuộc)

Hatch tự dựng "mission control" bằng Bubble Tea/lipgloss (như k9s/lazygit): **một process, 4 pane nội bộ** (`hatch board`): BOARD + LIVE (transcript) + ACTIVITY (ledger) + CHAT (bus, soạn được). Phím: `tab` đổi pane, `↑/↓` di/scroll, `f` follow ticket, `r` chạy run, `c` đổi channel, `i` soạn chat, `q` thoát.
```
┌ board ───────────┬ active runs ─────────────┐
│ backlog 3        │ T-101 codex   ▮▮▮ streaming│
│ in-progress 2    │ T-102 claude  ▮▮  streaming│
│ review 1         ├──────────────────────────┤
│ done 7           │ ledger feed (live tail)   │
├──────────────────┤ bus #design (live)        │
│ presence/WIP     │                           │
└──────────────────┴──────────────────────────┘
```
- Live agent output lấy từ **transcript** mỗi run (xem dưới) — TUI tail file, không cần điều khiển terminal khác.
- **Không cần tmux/Zellij**, chạy mọi nơi (kể cả SSH đơn). Đây là mặc định khuyến nghị.
- Đối ứng người: màn hình "mission control" của EM.

## Tầng B — Terminal multiplexer (tmux / Zellij) — tùy chọn

Khi muốn **mỗi agent một pane THẬT**, side-by-side, thấy UI native của agent (nhất là agent có TUI riêng hoặc stream rất nhiều) → Hatch điều phối một multiplexer:
- `hatch watch --mux=tmux` (hoặc `--mux=zellij`): tạo session, **mỗi run một pane**, đặt `hatch run <ticket>` vào pane đó; layout tự chia.
- `hatch run <ticket> --mux=tmux`: mở/split một pane cho run này.
- Cơ chế: shell ra `tmux new-session/split-window`/`zellij action new-pane`. Agent vốn đã là **process riêng** (orchestrator spawn) — multiplexer chỉ cấp cho mỗi process một khung nhìn.
- Đúng trực giác của bạn: **multiple TUI = tmux/Zellij**. Đây là tầng "phòng điều khiển nhiều màn hình" cho lúc theo dõi sát.

## Nền chung: transcript mỗi run (cần có trước)

Cả hai tầng dựa trên việc **ghi lại output thô** mỗi run (hiện orchestrator mới ghi tóm tắt vào ledger):
```
.hatch/runs/<ticket>/<ts>-<agent>.log     # stdout+stderr thô, append realtime
```
- Orchestrator stream ra: (a) terminal gọi lệnh, (b) file transcript, (c) tóm tắt + cost vào ledger.
- `hatch logs <ticket> [--follow]` — tail/replay transcript (như `kubectl logs -f`).
- Transcript là artifact replay được; TUI tầng A đọc nó; tmux tầng B hiển thị trực tiếp.

## Remote / CI (không có terminal tương tác)
Không tmux, không TUI. Quan sát qua: transcript file, `hatch standup`/`status`/`workload`, ledger, và PR. Hatch không phụ thuộc terminal tương tác — đó là lý do transcript + artifact là nền, còn TUI/tmux chỉ là *cách nhìn*.

## Đề xuất triển khai
1. **Transcript** mỗi run (`.hatch/runs/…`) + `hatch logs --follow` — nền cho mọi cách nhìn.
2. **TUI tầng A** mở rộng từ `hatch board`: thêm pane active-runs (tail transcript) + ledger/bus feed.
3. **Mux tầng B**: `--mux=tmux|zellij` cho `run`/`watch`.

Thứ tự: transcript trước (rẻ, dùng được ngay qua `logs`), rồi TUI, rồi mux.
