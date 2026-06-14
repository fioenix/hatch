# 05 — Workflow

> **Workflow là template, không phải luật cứng.** Mọi thứ trong doc này là **mặc định** mà Hatch ship sẵn. Người dùng có toàn quyền **thiết kế lại workflow ở từng project** — đổi lane, transition, gate, ceremony — qua `.hatch/workflow.yaml` (xem [spec/workflow](../spec/workflow.schema.md)). Board lanes, điều kiện chuyển trạng thái, và gate đều *đọc từ* file này; sửa file = đổi quy trình, không cần sửa code.

Mặc định, Hatch ghép hai chuẩn product development: **Agile** (vòng lặp, ticket, ceremonies) và **spec-driven** (PRD → design → tasks, kiểu Kiro). Cả hai đều ánh xạ về artifact file + chuyển trạng thái board.

## Lifecycle tổng (mặc định)

```
CHARTER ─► SPEC ─► BACKLOG ─► SPRINT ─► IN-PROGRESS ─► REVIEW ─► DONE ─► RETRO
   │        │         │          │           │            │        │       │
  L0     Architect Conductor  Conductor   Workers     Reviewer  merge  Conductor
                                                                       (+compaction)
```

1. **Charter** — định mission/ràng buộc (L0). Hiếm khi đổi.
2. **Spec** — với feature lớn, Architect viết spec-driven: PRD → Design → Tasks (xem dưới).
3. **Backlog** — Conductor bẻ spec/epic thành ticket có `role`, `priority`, `depends_on`.
4. **Sprint** — Conductor chọn tập ticket cho chu kỳ, đẩy lên đầu backlog.
5. **In-Progress** — workers claim & code (xem [protocol](03-coordination-protocol.md)).
6. **Review** — Reviewer gác DoD.
7. **Done** — merge sau human gate; ticket vào `done/`.
8. **Retro** — Conductor đúc kết, **nén ledger**, cập nhật SSOT nếu học được điều mới.

## Spec-driven cho feature lớn (mượn Kiro)

Một epic lớn không vào thẳng backlog mà qua 3 artifact, mỗi cái là một cổng:

```
product/epics/E-12/
├── prd.md       # WHY + WHAT: vấn đề, user, yêu cầu, tiêu chí chấp nhận
├── design.md    # HOW: kiến trúc, data model, API, trade-off, ADR
└── tasks.md     # ticket breakdown → Conductor sinh ra board/backlog/*
```

- `prd.md` do Conductor/PO + Architect. Gate: human duyệt PRD.
- `design.md` do Architect. Gate: review design trước khi code.
- `tasks.md` → Conductor sinh ticket. Mỗi ticket trỏ ngược về `design.md` qua `context_refs` (đây là L2 của nó).

Feature nhỏ/bug đi tắt: tạo ticket thẳng vào backlog, bỏ qua PRD/Design.

## Ceremonies → cơ chế async

Mỗi ceremony Agile có một đối ứng máy trong Hatch. Không họp đồng bộ — thay bằng artifact.

| Ceremony người | Đối ứng Hatch | Sinh ra |
|---|---|---|
| Sprint Planning | Conductor chạy pha PLAN | tickets có chủ trong backlog/ |
| Daily Standup | **Ledger digest** — Conductor tóm tắt ledger từ lần trước | trạng thái + blocker nổi lên |
| Sprint Review | Tổng hợp `done/` trong chu kỳ | bản tổng kết / changelog |
| Retro | Conductor đọc ledger → bài học | cập nhật SSOT/protocol + nén ledger |
| Backlog Grooming | Conductor xếp lại ưu tiên + dependency | backlog gọn |

### Standup = ledger digest (cơ chế token quan trọng)

Thay vì mỗi agent đọc lại toàn bộ lịch sử (tốn token), Conductor định kỳ đọc ledger thô và viết một **digest ngắn**: ai đang làm gì, gì xong, gì block. Digest này là "trí nhớ chung nén lại" — agent đọc digest thay vì lịch sử đầy đủ. Giống standup người: 15 phút thay vì đọc hết Jira.

## Định nghĩa "sprint" linh hoạt

Hatch không ép thời lượng. "Sprint" = một tập ticket Conductor cam kết cho một chu kỳ (có thể là một phiên làm việc, một ngày, hay một tuần). Cái cần là **ranh giới rõ** để có điểm review + retro + nén ledger.

## Workflow-as-template: thiết kế lại per-project

Lifecycle trên là khung **mặc định**. Người dùng chọn một **template có sẵn** hoặc **viết workflow riêng** trong `.hatch/workflow.yaml`. Hatch ship các template:

- **`scrum`** (mặc định): sprint cố định + retro mỗi sprint.
- **`kanban`:** không sprint, WIP limit theo lane, pull liên tục.
- **`spec-first`:** mọi epic bắt buộc qua PRD→Design→Tasks trước khi vào backlog.
- **`lite`:** chỉ `todo → doing → done` cho project nhỏ/cá nhân.

`workflow.yaml` khai báo (xem [spec/workflow](../spec/workflow.schema.md)):
- **lanes** — các trạng thái (= thư mục trong `board/`).
- **transitions** — chuyển nào hợp lệ, ai (vai nào) được làm.
- **gates** — điều kiện pass khi qua một transition (test, DoD, human approval).
- **ceremonies** — sự kiện định kỳ (planning, standup-digest, retro) và ai chủ trì.

Ví dụ một transition tùy biến:
```yaml
transitions:
  - from: in-progress
    to: review
    by: [implementer]
    gates: [tests-pass, lint-clean, handoff-note]
```

Đổi quy trình = sửa `workflow.yaml`. Board, protocol, và orchestrator đều đọc từ đây — Hatch cung cấp khung + cơ chế, *chuẩn cụ thể* luôn là cấu hình per-project, không hard-code.
