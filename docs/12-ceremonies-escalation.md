# 12 — Ceremonies, escalation & decisions

Những nghi thức làm cho Hatch hành xử như một squad người thật, không chỉ một hàng đợi task.

## Ceremonies sống

`workflow.yaml` *khai báo* ceremonies; `hatch ceremony` *chạy* chúng.

| Lệnh | Mô phỏng | Làm gì |
|---|---|---|
| `hatch ceremony standup [--days N] [--post]` | standup hằng ngày | Digest theo agent từ ledger (đã làm gì) + blockers từ lane blocked; mặc định post vào `#standup`. |
| `hatch ceremony retro [--write]` | retrospective cuối chu kỳ | Tổng kết: done / blocks / gate failures / decisions; liệt kê **ứng viên đề bạt KB→SSOT** (learnings). `--write` lưu `ledger/retro-<date>.md`. |
| `hatch ceremony planning [--dry-run]` | sprint planning | Spawn Conductor bẻ việc (như `hatch plan`). |
| `hatch ceremony demo [--post]` | sprint review / demo | Trình diễn việc ở lane terminal; post `#demo`. |
| `hatch ceremony grooming` | backlog refinement | Soi ticket backlog thiếu role/priority/acceptance. |

Chủ trì (`chair`) lấy từ `workflow.yaml > ceremonies.<name>.by`, mặc định `human:facilitator`.

## Escalation / on-call

Như dev kẹt thì gọi senior. Đích escalate = `registry.policy.escalate_to` → nếu trống thì Conductor đầu tiên → `human:lead`.

- **Thủ công:** `hatch escalate <ticket> --why "..."` → ghi ledger `action: escalate` + post `#escalations` (tag đích danh `@target`).
- **Tự động:** khi một ticket **fail gate ≥ 2 lần**, workflow engine tự escalate một lần (không spam) — xem [governance](06-governance.md). "Blocked quá lâu" có thể quét bằng cron + `hatch escalate`.

## On-call & incidents

Như đội có lịch trực: `hatch oncall set --rotation a,b,c` định nghĩa vòng trực, `hatch oncall` xem ai đang trực, `hatch oncall rotate` bàn giao pager (báo `#oncall`). **Escalation tự nhắm người đang trực** trước (rồi mới tới `policy.escalate_to` → Conductor → `human:lead`).

Template workflow **`incident`** (`hatch init -w incident`): `detected → triage → mitigating → resolved → postmortem`, gate `fix-verified` (test) + `postmortem-written`. Ghép on-call + escalation tự động khi kẹt = quy trình ứng cứu sự cố đầy đủ.

## Presence / capacity

`hatch presence` cho thấy ai `available/busy/paused/offline` + tải WIP; `hatch presence set <agent> --status …`. Khâu phân việc (`run`/`watch`/pairing) **bỏ qua agent paused/offline và ưu tiên agent ít tải nhất dưới WIP** — như lead giao cho người đang rảnh.

## Mob

`hatch mob <ticket> --agents a,b,c` mở rộng pairing cho 3+ agent: **driver xoay vòng mỗi vòng**, còn lại navigate; kết thúc sớm khi đa số navigator `READY`.

## Decisions → ADR

Khép vòng từ họp tới tri thức của record: trong `hatch convene`, khi một lượt mở đầu bằng `DECISION:`, Hatch **tự ghi một ADR** vào `kb/decisions/` (status `accepted`, link tới thread họp) + entry ledger. Quyết định không bốc hơi trong chat — nó thành tri thức tra cứu được (xem [knowledge-base](09-knowledge-base.md)).

```
convene (#meet) ── "DECISION: dùng CSV streaming" ──► kb/decisions/ADR-00X + ledger note
```

## Pairing (driver/navigator)

Hai agent cùng một ticket như pair programming: **driver** triển khai từng bước nhỏ, **navigator** soi lỗi/rủi ro và gợi ý bước kế — luân phiên qua thread `pair-<ticket>`.

```bash
hatch pair T-001 --driver codex --navigator claude-code --rounds 3 [--claim]
```

Mỗi vòng: driver chạy một lượt (thấy feedback navigator gần nhất) → ghi thread → navigator review lượt đó → ghi thread. Navigator mở đầu `READY` ⇒ kết thúc sớm (đủ tốt để chuyển review). `--dry-run` xem cấu trúc lượt mà không spawn. Bắt buộc driver ≠ navigator.

## Vì sao quan trọng

Ba mảnh này biến vòng đời từ "giao task → chạy" thành nhịp một đội thật: **đồng bộ hằng ngày** (standup), **học từ chu kỳ** (retro → đề bạt SSOT), **gọi cứu viện khi kẹt** (escalation), và **chốt quyết định thành ADR**. Tất cả vẫn append-only, auditable.
