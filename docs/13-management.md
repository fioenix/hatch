# 13 — Management plane (Founder/CEO/CTO view)

> **Trạng thái: PHẦN LỚN ĐÃ IMPLEMENT** — `hatch workload · perf · cost · budget · report` chạy được; cost track-only (không auto-pause, theo quyết định ở [17](17-pre-implementation.md)). Doc này là thiết kế gốc của lớp quản lý vận hành. Mọi số liệu **derive từ artifact đã có** (ledger + board + bus) — không thêm hệ thống tracking song song. Một người ở vai Founder/CEO/CTO cần ba thứ: **workload** (đội đang tải bao nhiêu), **performance** (làm tốt không), **budget/lương** (tốn bao nhiêu, có vượt không).

## 0. Nền dữ liệu: mọi metric đến từ ledger

Không có DB phân tích riêng. Ledger append-only đã ghi who/what/when/why; board cho trạng thái; bus cho đối thoại. Mọi chỉ số tính ra từ đó:

| Chỉ số | Nguồn (derive) |
|---|---|
| Cycle time (1 ticket) | ledger: `claim` → `done` |
| Lead time | ticket `created` → `done` |
| Throughput | số `done` / chu kỳ |
| Rework | số `review → in-progress` (changes-requested) |
| Gate pass rate | `gate` result passed / (passed+failed) |
| Escalations | số entry `escalate` |
| KB đóng góp | số mục `kb/` theo author |
| Review load | số `review` theo agent |

Một ticket nên có thêm field tùy chọn cho ước lượng:
```yaml
estimate: 3        # điểm hoặc giờ (đơn vị do team chọn trong charter)
```
"Actual" không cần nhập tay — tính từ ledger. **Estimate vs actual** = học để ước lượng tốt hơn (đưa vào retro).

## 1. Workload management

Mở rộng presence + WIP đã có thành bức tranh tải toàn đội.

- **`hatch workload`** — bảng theo agent: WIP hiện tại / WIP limit, queue (ticket gán nhưng chưa làm), throughput chu kỳ, avg cycle time, trạng thái presence. Gắn cờ **quá tải** (WIP ≥ limit) và **rảnh** (idle) để cân tải.
- **Burndown/burnup** — `hatch workload --burn`: còn lại vs đã done theo thời gian trong sprint (đọc ledger).
- **Cân tải**: Conductor/`watch` đã ưu tiên agent ít tải; workload view cho người quyết định can thiệp thủ công.

Human analogue: bảng capacity của EM — ai ngập việc, ai rảnh, sprint có kịp không.

## 2. Performance management

Bảng điểm vận hành mỗi agent, **tính từ ledger** — không phải "đánh giá con người".

- **`hatch perf [<agent>]`** — scorecard: done, avg cycle time, rework rate, gate pass rate, escalations gây ra, reviews thực hiện, KB đóng góp, decisions chốt.
- **Xu hướng**: so scorecard giữa các chu kỳ (đang lên/xuống) — đầu vào cho 1:1.
- **1:1 / feedback**: ghi nhận định kỳ dạng KB note (`kb/perf/<agent>-<cycle>.md`) hoặc DM qua bus — phản hồi cụ thể + mục tiêu kỳ tới.
- **Thăng cấp năng lực**: registry có thể nâng/giảm vai một agent dựa trên scorecard (vd gate pass rate cao → cho làm reviewer).

> **Cảnh báo trung thực (Goodhart).** Agent không phải người; đây là **chỉ số chất lượng vận hành**, không phải "thành tích cá nhân". Tối ưu mù theo một metric (vd throughput) sẽ phá chất lượng. Dùng để phát hiện bất thường + điều chỉnh phân vai/prompt/SSOT, không để "phạt".

## 3. Budget & "lương" (cost control)

Tương tự Paperclip (agents có lương + ngân sách), nhưng file-based và derive được.

### Chi phí thật
Mỗi headless run trả về cost (Claude `total_cost_usd`, Codex/Gemini usage). Orchestrator ghi cost vào ledger mỗi run:
```markdown
## <ts> · codex · T-123
- action: progress
- cost_usd: 0.42
- tokens: 18450
```

### "Lương" = ngân sách cấp cho mỗi agent
Trong registry:
```yaml
agents:
  - id: claude-code
    rate_per_mtok: 15.0     # giá tham chiếu (USD / 1M token)
    budget_usd: 200         # "lương"/trần chi mỗi chu kỳ
policy:
  team_budget_usd: 1000     # trần toàn đội / chu kỳ
```

### Theo dõi + trần cứng + auto-pause
- **`hatch budget`** — burn vs cap theo agent + toàn đội; cảnh báo ở 80%.
- **`hatch cost <ticket>`** — tổng chi cho một ticket (sum cost ledger entries).
- **Trần cứng**: khi một agent (hoặc đội) vượt budget → tự `presence set <agent> paused` + báo `#budget`/escalate. Vì capacity-aware assignment đã bỏ qua agent paused, **vượt ngân sách = tự ngừng giao việc** cho agent đó. (Đây là chỗ presence + budget khớp nhau đẹp.)

### Góc CEO
- **Payroll** = Σ `budget_usd` (cam kết). **Spend thực** = Σ cost ledger. **ROI** = throughput (hoặc story point done) / spend.
- Báo cáo: "tuần này đội tiêu $X/$Y ngân sách, ship N ticket, $/ticket = …".

## 4. Status report ra ngoài (stakeholder)

Khác standup (nội bộ, chi tiết) — **`hatch report`** sinh tóm tắt điều hành: trạng thái board (done/in-flight/blocked), throughput + burndown, budget burn, rủi ro (escalations, ticket quá hạn), quyết định lớn (ADR mới). Một trang cho Founder/đầu tư/đối tác. Có thể post `#leadership` hoặc xuất file.

## Vì sao quan trọng
Ba lớp này biến Hatch từ "đội tự chạy" thành "đội **quản trị được**": người đứng đầu thấy **đội tải bao nhiêu** (workload), **chạy tốt không** (performance), **tốn bao nhiêu & có trong ngân sách không** (budget) — tất cả derive từ cùng một ledger append-only, nên minh bạch và audit được. Xem thêm tổ chức & uỷ quyền ở [14-org-and-cadence](14-org-and-cadence.md).
