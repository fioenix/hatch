# Spec — Ticket

Một đơn vị việc. File Markdown trong `.hatch/board/<lane>/T-<id>.md`. **Vị trí thư mục = trạng thái.** Frontmatter = metadata; body = mô tả + acceptance.

## Frontmatter

```yaml
---
id: T-123                      # duy nhất, ổn định
title: "Export báo cáo ra CSV"
status: in-progress            # backlog | in-progress | review | done | blocked
role: implementer              # vai chịu trách nhiệm ở giai đoạn hiện tại
assignee: codex                # agent đang giữ (rỗng nếu chưa claim)
priority: P1                   # P0 | P1 | P2 | P3
epic: E-12                     # epic/spec gốc (rỗng nếu standalone)
depends_on: [T-100, T-101]     # phải done hết mới được claim
branch: "hatch/T-123-export-csv" # branch-per-ticket
context_refs:                  # L2 — đọc on-demand khi claim, KHÔNG nhúng sẵn
  - context/tech/reporting.md
  - product/epics/E-12/design.md
claim:                         # cơ chế lock (xem coordination-protocol)
  agent: codex
  ts: "2026-06-14T09:12:03+07:00"
dod:                           # bổ sung ngoài DoD mặc định
  - "Có test cho ký tự đặc biệt và unicode"
created: "2026-06-14T08:00:00+07:00"
updated: "2026-06-14T09:12:03+07:00"
---
```

## Body

```markdown
## Bối cảnh
Người dùng cần xuất báo cáo doanh thu ra CSV để mở bằng Excel.

## Yêu cầu
- Endpoint `GET /reports/{id}/export?format=csv`
- Encoding UTF-8 BOM (Excel đọc đúng tiếng Việt)
- Stream, không load hết vào RAM

## Acceptance
- [ ] File mở đúng trên Excel, không lỗi font
- [ ] Báo cáo 1 triệu dòng không OOM
- [ ] Test pass

## Handoff notes
<!-- mỗi lần đổi assignee/role, thêm 1 mục: đã làm gì, còn gì, cần gì -->
- 2026-06-14 architect→implementer: design ở E-12/design.md §4; dùng csv streaming có sẵn ở utils/stream.
```

## Quy tắc

- **Đổi trạng thái** = `git mv` sang lane mới + cập nhật `status`/`updated` + entry ledger. Một thay đổi = một commit.
- **Claim** = set `assignee` + `claim` + `git mv` vào `in-progress/`, push ngay (push thắng = lock thắng).
- **`context_refs` là L2:** agent chỉ đọc khi nhận ticket → đây là điểm kiểm soát token chính.
- **`depends_on`:** không claim được khi còn dependency chưa `done`.
- **Handoff notes** bắt buộc khi đổi `assignee` — context tối thiểu để người sau không đọc lại toàn bộ.
- **Ranh giới:** không sửa file ngoài `context_refs` đã khai (Reviewer gác). Mở rộng scope → cập nhật ticket trước.

## Vòng đời lane

```
backlog/ → in-progress/ → review/ → done/
                │            ↑
                └─ blocked/ ─┘
```

## Validate (pre-commit / orchestrator)

- `id` duy nhất toàn board.
- `status` khớp lane chứa file.
- Nếu có `claim.agent` → vai `role` phải nằm trong `roles` của agent đó (registry).
- `no-self-review`: khi vào `review/`, `assignee` (reviewer) ≠ implementer trước đó.
