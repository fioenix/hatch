# 16 — Document & report templates/specs

> **Trạng thái: THIẾT KẾ (chưa implement).** Như đội người, agent viết **rất nhiều tài liệu** trong lúc làm (PRD, design doc, ADR, RFC, postmortem, runbook, test plan, release notes, status report…). Mỗi loại cần **template + spec chuẩn theo một framework**, và **user override được** — đúng triết lý "template per-project" như workflow.

## Mô hình: doc type → framework → template (sửa được)

Hatch ship một bộ loại tài liệu, mỗi loại gắn một framework công nhận rộng rãi, dưới dạng template có spec. User copy & sửa per-project.

| Doc type | Framework mặc định | Lưu ở |
|---|---|---|
| `adr` | MADR / Nygard ADR | `kb/decisions/` |
| `rfc` | RFC (đề xuất + thảo luận) | `kb/` hoặc `docs/rfc/` |
| `prd` | Product Requirements Doc | `context/product/` |
| `design` | Design doc (kiểu Google) | spec feature / `context/tech/` |
| `requirements` | **EARS** (khớp Kiro) | spec feature |
| `postmortem` | Blameless postmortem (Google SRE) | `kb/` |
| `runbook` | Ops runbook | `context/tech/` |
| `test-plan` | Test plan | ticket / spec |
| `release-notes` | Keep a Changelog + SemVer | `docs/` |
| `status-report` | Exec report (xem [13](13-management.md)) | post `#leadership` |
| `tech-doc` | **Diátaxis** (tutorial/how-to/reference/explanation) | `context/` |

## Lưu trữ & cấu trúc

```
.hatch/templates/docs/
├── adr.md            # template + spec (frontmatter khai required sections/fields)
├── design.md
├── postmortem.md
└── ...               # user thêm/sửa tự do
```

Mỗi template có frontmatter **spec** + thân **scaffold**:
```markdown
---
doc-type: adr
framework: MADR
required-sections: [Context, Decision, Consequences, Alternatives]
required-frontmatter: [id, title, status, date]
---
# {{title}}

## Context
<!-- vấn đề + ràng buộc -->
## Decision
## Consequences
## Alternatives
```

## CLI

```bash
hatch doc types                       # liệt kê doc type + framework
hatch doc new design --title "Export API" [--ticket T-123]   # scaffold từ template
hatch doc lint <file>                 # kiểm spec: đủ required sections + frontmatter?
hatch doc lint --all                  # quét toàn repo
```

- **`doc new`** sinh file đúng nơi (bảng trên) từ template, điền placeholder, mở (Obsidian nếu KB-mode).
- **`doc lint`** = "spec gate": file thiếu section/field bắt buộc ⇒ fail. Có thể gắn vào **DoD gate** của workflow (vd ticket loại design phải `doc lint` pass) hoặc pre-commit.

## Tích hợp với phần đã có
- **KB**: `hatch kb add --type decision` dùng template `adr.md` (gộp, không trùng cơ chế).
- **Spec-first workflow**: artifact `requirements/design/tasks` dùng template `requirements` (EARS) + `design`.
- **Compiler/role**: charter/role nhắc agent "khi viết <type>, dùng `hatch doc new <type>` và tuân spec".
- **Management**: `status-report` template chính là output của `hatch report` ([13](13-management.md)).

## Custom hoá (yêu cầu cốt lõi)
- User sửa thẳng `templates/docs/<type>.md` hoặc thêm type mới (vd `architecture-review`, `security-review`).
- Đổi framework = đổi nội dung template + `required-sections`. Hatch chỉ cưỡng chế cái template khai báo, không hardcode framework.
- Bộ ship sẵn chỉ là **điểm xuất phát**, hệt như 8 workflow template.

## Vì sao quan trọng
Tài liệu agent viết ra sẽ **nhất quán, đúng chuẩn, lint được** — thay vì mỗi agent một kiểu. Cộng với Obsidian KB ([15](15-obsidian-kb.md)), đội có một "second brain" có cấu trúc: tài liệu đúng spec, liên kết wikilink, tra cứu theo đồ thị.
