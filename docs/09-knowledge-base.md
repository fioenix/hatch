# 09 — Knowledge Base

Agents trong Hatch **không chia sẻ bộ nhớ** — mỗi con một process, hết phiên là quên. Nhưng một đội hiệu quả cần *trí nhớ chung tích lũy*. KB là câu trả lời: một **bộ não chung trên git** mà mọi agent đọc *và* ghi. Đối ứng người: wiki của đội (Confluence/Notion) + kho ADR.

## Vì sao cần KB (tách khỏi SSOT và Ledger)

| Kho | Trả lời câu hỏi | Vòng đời |
|---|---|---|
| `context/` (SSOT) | "Chuẩn/định hướng là gì?" | Ổn định, human-curated, **compile** vào agent |
| `kb/` (Knowledge Base) | "Ta đã biết/quyết gì về thứ này?" | Tăng dần, mọi agent ghi, tra cứu on-demand |
| `ledger/` | "Ai đã làm gì, khi nào, vì sao?" | Append-only, sự kiện theo thời gian |

Không có KB, tri thức học được trong một ticket sẽ **bốc hơi** khi agent kết thúc — agent sau lại dò lại từ đầu (tốn token, dễ sai khác). KB biến trải nghiệm rời rạc thành tài sản chung.

## Cấu trúc

```
.hatch/kb/
├── index.md            # mục lục có tag → tra cứu nhanh, ít token
├── decisions/          # ADR: quyết định kiến trúc + bối cảnh + lý do + hệ quả
│   └── ADR-007-csv-streaming.md
├── domain/             # tri thức nghiệp vụ tích lũy (rule, thuật ngữ, ràng buộc thật)
├── learnings/          # bài học, gotcha, pitfall đã gặp + cách xử lý
└── .meta.json          # tag/owner/updated/links cho từng mục (để truy vấn)
```

Mỗi mục KB là một file Markdown ngắn có frontmatter:

```yaml
---
id: ADR-007
type: decision            # decision | domain | learning
title: "Dùng CSV streaming cho export"
tags: [reporting, performance, export]
related: [T-123, context/tech/reporting.md]
author: kiro              # agent hoặc human:<tên>
created: "2026-06-14T10:00:00+07:00"
status: accepted          # cho ADR: proposed | accepted | superseded
---
```

## Đọc — input chung

Khi nhận ticket, agent **truy vấn KB theo tag/`related`** (qua `index.md`) và chỉ nạp các mục liên quan như một phần L2. Không nạp cả KB. Đây là điểm tiết kiệm token: thay vì suy diễn lại "tại sao reporting làm thế này", agent đọc `ADR-007` trong vài trăm token.

Phase 1: agent đọc `index.md` rồi mở mục cần. Phase 2/3: `hatch kb query <tag>` trả về tập mục liên quan.

## Ghi — output chung

Agent **ghi vào KB khi học được điều đáng giữ**, không phải mọi lúc. Quy ước (đưa vào role context):

- Ra một **quyết định kiến trúc** → tạo ADR trong `decisions/`.
- Phát hiện một **gotcha/pitfall** tốn thời gian → ghi `learnings/`.
- Làm rõ một **rule nghiệp vụ** chưa có ở đâu → ghi `domain/`.
- Cập nhật `index.md` + `.meta.json` (Phase 2/3 tự động hóa bước này).

Ghi KB là một phần của **Definition of Done** cho ticket có hàm lượng tri thức cao (xem [protocol DoD](03-coordination-protocol.md#definition-of-done-dod)).

## KB vs SSOT — đường thăng cấp

KB là nơi tri thức *nảy mầm*; SSOT là nơi tri thức *đã chín thành chuẩn*. Trong **Retro** ([workflow](05-workflow.md)), Conductor rà KB: mục nào đã thành chuẩn chính thức → **đề bạt** (promote) vào `context/` (SSOT) để compile cho mọi agent. Ví dụ: một `learning` về cách viết test lặp lại nhiều lần → nâng thành rule trong `context/tech/`.

```
learnings/ (cá biệt) ──promote──► context/tech/ (chuẩn, compile vào agent)
decisions/ (ADR)     ──reference─► agent tra cứu khi liên quan
```

## Tránh phình & nhiễu

- **Một mục một ý.** Mục KB ngắn, có tag. Không nhật ký dài dòng (đó là việc của ledger).
- **Chống trùng:** trước khi ghi, agent tra `index.md`; nếu đã có mục gần giống → cập nhật/đính kèm thay vì tạo mới.
- **Khử lỗi thời:** ADR cũ bị thay → đánh `status: superseded` + trỏ tới ADR mới (không xóa, giữ lịch sử quyết định).
- **Retro dọn KB:** gộp mục trùng, archive mục hết giá trị.

## Quyền ghi & governance

- Mọi vai được ghi KB, nhưng **đề bạt KB→SSOT** cần Architect/Conductor (giống review trước khi lên wiki chính thức).
- Ghi KB cũng để lại entry ledger (`action: note`, link tới mục KB) → vẫn truy vết được ai thêm tri thức gì.
- KB nằm trong git → review được qua PR như mọi thay đổi khác.
