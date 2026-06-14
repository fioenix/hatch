# 14 — Org chart, delegation & cadence

> **Trạng thái: THIẾT KẾ (chưa implement).** Brainstorm cho ba mảnh vận hành còn thiếu: **org-chart + uỷ quyền**, **phụ thuộc liên đội/bên ngoài**, và **nhịp/heartbeat**. Nối tiếp [13-management](13-management.md).

## 1. Org chart & Delegation of Authority (DoA)

Hiện registry là phẳng (vai + agent). Đội thật có **đường báo cáo** và **hạn mức thẩm quyền**. Bổ sung vào registry:

```yaml
roles:
  - id: conductor
    reports_to: ""              # gốc cây (CEO/Founder)
    authority:
      can_approve: true         # được duyệt merge
      merge_limit: any          # phạm vi được merge
      budget_authority_usd: 1000 # tự duyệt chi tới mức này
      decision_scope: [arch, hiring, budget]
  - id: implementer
    reports_to: conductor
    authority:
      can_approve: false
      budget_authority_usd: 0
```

Hệ quả:
- **Escalation đi theo cây báo cáo**: vượt thẩm quyền của vai hiện tại → chuyển lên `reports_to` (thay vì luôn về Conductor). Kết hợp on-call ở [12](12-ceremonies-escalation.md).
- **Gate `human-merge`/`budget`** kiểm tra DoA: hành động vượt `budget_authority_usd` hoặc ngoài `decision_scope` ⇒ bắt buộc escalate, không tự quyết.
- **`hatch org`** — in cây tổ chức + ma trận thẩm quyền (đối ứng "Delegation of Authority Matrix" của một công ty thật).

Human analogue: ai báo cáo cho ai, ai được ký tới mức nào — chống việc một agent tự quyết vượt quyền.

## 2. Phụ thuộc liên đội / bên ngoài

`depends_on` hiện chỉ trỏ ticket nội bộ. Thực tế đội còn **chờ bên ngoài**: vendor, đội khác, phê duyệt của người, một repo khác.

Bổ sung frontmatter ticket:
```yaml
blocked_by_external:
  - what: "Khoá API từ nhà cung cấp thanh toán"
    owner: "human:vendor-x"
    eta: "2026-06-20"
    status: waiting        # waiting | received
```

- Không tự gỡ — phải có người/sự kiện đánh dấu `received`.
- Hiện trong `hatch status`/`workload`/standup như **rủi ro tiến độ** (khác blocker nội bộ).
- **Cross-repo (tương lai):** dependency trỏ tới ticket ở một workspace Hatch khác (multi-repo, ngoài phạm vi bản đầu).

## 3. Heartbeat / cadence (scheduler)

Ceremonies đã khai báo `trigger` (daily, end-of-sprint, cron) nhưng cần thứ **đánh thức** đội định kỳ — như Paperclip heartbeat.

- **`hatch tick`** — chạy MỘT nhịp: thực thi ceremony tới hạn (standup digest, rotate on-call nếu tới lịch), dispatch việc claimable (`watch` một pass tôn trọng WIP/presence), kiểm budget (auto-pause nếu vượt). Idempotent, để gọi từ **cron / GitHub Actions / systemd timer**.
- **`hatch daemon`** (tuỳ chọn) — vòng lặp gọi `tick` theo chu kỳ, cho máy chạy thường trú.
- Mốc lần chạy lưu ở `.hatch/.schedule.json` (last-run mỗi ceremony) để biết cái gì "tới hạn".

> **Lưu ý môi trường.** Trong môi trường remote ephemeral, không có tiến trình thường trú — cadence do **bên ngoài** kéo (cron của CI gọi `hatch tick`). Thiết kế nhắm "stateless tick gọi được từ scheduler bất kỳ", không phụ thuộc daemon.

## 4. Mảnh nhỏ còn lại

- **Estimate/time-tracking**: field `estimate` trên ticket + cycle time derive từ ledger (chi tiết ở [13 §0](13-management.md)). Estimate vs actual đưa vào retro.
- **Spike/research**: một `kind: spike` cho ticket điều tra có time-box (đánh dấu để không tính vào throughput sản phẩm).

## Thứ tự đề xuất implement
1. Cost capture trong ledger + `hatch budget`/`cost` + auto-pause (giá trị CEO cao nhất, nối sẵn presence).
2. `hatch workload` + `hatch perf` (đọc ledger — rẻ, không đụng spawn).
3. Org-chart + DoA trong registry + escalation theo cây.
4. External dependency field + hiển thị rủi ro.
5. `hatch tick` (cadence) + `hatch report` (stakeholder).

Mỗi mục đều **derive từ artifact sẵn có** nên rủi ro thấp, chủ yếu là đọc + trình bày + vài field cấu hình.
