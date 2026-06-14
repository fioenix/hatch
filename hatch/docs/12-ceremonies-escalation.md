# 12 — Ceremonies, escalation & decisions

Những nghi thức làm cho Hatch hành xử như một squad người thật, không chỉ một hàng đợi task.

## Ceremonies sống

`workflow.yaml` *khai báo* ceremonies; `hatch ceremony` *chạy* chúng.

| Lệnh | Mô phỏng | Làm gì |
|---|---|---|
| `hatch ceremony standup [--days N] [--post]` | standup hằng ngày | Digest theo agent từ ledger (đã làm gì) + blockers từ lane blocked; mặc định post vào `#standup`. |
| `hatch ceremony retro [--write]` | retrospective cuối chu kỳ | Tổng kết: done / blocks / gate failures / decisions; liệt kê **ứng viên đề bạt KB→SSOT** (learnings). `--write` lưu `ledger/retro-<date>.md`. |
| `hatch ceremony planning [--dry-run]` | sprint planning | Spawn Conductor bẻ việc (như `hatch plan`). |

Chủ trì (`chair`) lấy từ `workflow.yaml > ceremonies.<name>.by`, mặc định `human:facilitator`.

## Escalation / on-call

Như dev kẹt thì gọi senior. Đích escalate = `registry.policy.escalate_to` → nếu trống thì Conductor đầu tiên → `human:lead`.

- **Thủ công:** `hatch escalate <ticket> --why "..."` → ghi ledger `action: escalate` + post `#escalations` (tag đích danh `@target`).
- **Tự động:** khi một ticket **fail gate ≥ 2 lần**, workflow engine tự escalate một lần (không spam) — xem [governance](06-governance.md). "Blocked quá lâu" có thể quét bằng cron + `hatch escalate`.

## Decisions → ADR

Khép vòng từ họp tới tri thức của record: trong `hatch convene`, khi một lượt mở đầu bằng `DECISION:`, Hatch **tự ghi một ADR** vào `kb/decisions/` (status `accepted`, link tới thread họp) + entry ledger. Quyết định không bốc hơi trong chat — nó thành tri thức tra cứu được (xem [knowledge-base](09-knowledge-base.md)).

```
convene (#meet) ── "DECISION: dùng CSV streaming" ──► kb/decisions/ADR-00X + ledger note
```

## Vì sao quan trọng

Ba mảnh này biến vòng đời từ "giao task → chạy" thành nhịp một đội thật: **đồng bộ hằng ngày** (standup), **học từ chu kỳ** (retro → đề bạt SSOT), **gọi cứu viện khi kẹt** (escalation), và **chốt quyết định thành ADR**. Tất cả vẫn append-only, auditable.
