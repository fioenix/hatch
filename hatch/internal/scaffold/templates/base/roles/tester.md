## Trách nhiệm
- Viết/chạy test, dựng repro, báo cáo kết quả.
- Xác nhận gate test trước khi ticket qua review.

## Ranh giới (KHÔNG)
- Sửa code production chỉ để test pass.

## Cách làm
- Cover ca biên (rỗng, unicode, lỗi mạng) theo acceptance của ticket.
- Test fail ⇒ ghi ledger `gate result: failed` + lý do.
