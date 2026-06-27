# Spec — Ledger

Sổ audit append-only. Mỗi thay đổi trạng thái có ý nghĩa = một entry. Trong git nên bất biến kép. Phục vụ standup digest, retro, điều tra sự cố, và chứng cứ audit.

## Tổ chức file

Một file / ngày: `.hatch/ledger/YYYY-MM-DD.md`. (Tùy chọn: thêm `.hatch/ledger/by-ticket/T-123.md` tổng hợp theo ticket — sinh tự động, không phải nguồn.)

## Entry — trả lời WHO · WHAT · WHEN · WHY · WHERE

```markdown
## 2026-06-14T09:12:03+07:00 · codex · T-123
- action: claim                       # claim | start | progress | handoff | review | done | block | unblock | revoke | note
- from: backlog/ → in-progress/       # chuyển lane (nếu có)
- why: sprint S-7, P1, dependency T-100 done
- branch: hatch/T-123-export-csv
- note: bắt đầu với csv streaming utils

## 2026-06-14T10:40:11+07:00 · codex · T-123
- action: handoff
- from: in-progress/ → review/
- to-role: reviewer
- why: code xong, test xanh
- handoff: endpoint ở reporting/export.go; test ở export_test.go; chưa cover file rỗng

## 2026-06-14T11:05:00+07:00 · claude-code · T-123
- action: review
- result: changes-requested
- why: thiếu test file rỗng (DoD)
- from: review/ → in-progress/
```

## Trường

| Trường | Bắt buộc | Ý nghĩa |
|---|---|---|
| timestamp (heading) | ✓ | ISO 8601 + offset (Asia/Ho_Chi_Minh) |
| agent (heading) | ✓ | WHO — agent (hoặc `human:<tên>`) |
| ticket (heading) | ✓* | WHERE — id ticket (`-` nếu sự kiện cấp hệ thống) |
| `action` | ✓ | WHAT — loại hành động (enum) |
| `from` | tùy | chuyển lane `A/ → B/` |
| `why` | ✓ | WHY — lý do; cốt lõi cho audit |
| `result` | tùy | với review/gate: approved \| changes-requested \| failed |
| `handoff` | ✓ khi action=handoff | context tối thiểu cho vai sau |
| `branch` / `note` | tùy | bổ sung |

## Action enum

`claim · start · progress · handoff · review · done · block · unblock · revoke · note · gate · escalate`

## Quy tắc

- **Append-only:** không sửa/xóa entry cũ. Sai → thêm entry `action: note` đính chính.
- **Một chuyển trạng thái = một entry = một commit** (cùng commit với `git mv` ticket).
- `why` không được rỗng — đây là giá trị audit. "vì sao" quan trọng hơn "cái gì" (cái gì đã có trong diff).
- `escalate` / sự cố nghiêm trọng → entry riêng, báo người ngay (xem [governance](../docs/06-governance.md)).

## Tiêu thụ

- **Standup digest:** Conductor đọc entry từ mốc trước → tóm tắt ngắn (ai làm gì, xong gì, block gì). Agent đọc digest thay vì toàn bộ ledger → tiết kiệm token.
- **Retro:** đọc cả chu kỳ → bài học → cập nhật SSOT/protocol.
- **Audit:** truy vết đầy đủ một ticket qua `by-ticket/` hoặc grep theo id.
