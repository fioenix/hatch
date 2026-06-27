# 15 — Obsidian như KB chính (qua CLI)

> **Trạng thái: THIẾT KẾ (chưa implement).** Mở rộng [09-knowledge-base](09-knowledge-base.md): ngoài MD thô tự tổ chức, cho phép dùng **Obsidian vault** làm KB chính, thao tác qua CLI. Lý do: Obsidian = markdown + wikilink + graph + tag + plugin — đúng "second brain" cho một đội, mà vault chỉ là thư mục .md nên CLI/agent thao tác trực tiếp được.

## Nguyên tắc: vault là file, nên CLI sở hữu được

Obsidian là GUI, **không có CLI chính thức**. Nhưng vault chỉ là folder markdown → Hatch thao tác trực tiếp (tạo/sửa/link/tag/index), và "tích hợp Obsidian" gồm:
1. **Tương thích định dạng** Obsidian (wikilink, tag, frontmatter, Dataview) để mở trong app là dùng được ngay.
2. **CLI quản lý vault** (`hatch kb …`) — nguồn ghi chính cho agent.
3. **Cầu nối app**: Obsidian URI (`obsidian://open?vault=…&file=…`) để mở note; tuỳ chọn plugin **Local REST API** cho tích hợp live khi app đang chạy.

## Cấu hình

Trong registry/charter:
```yaml
kb:
  mode: obsidian          # native | obsidian
  vault: ./kb             # đường dẫn vault (mặc định .hatch/kb là vault luôn)
  wikilinks: true         # related → [[wikilink]] thay vì đường dẫn
  dataview: true          # sinh frontmatter thân thiện Dataview
```
`mode: native` = hành vi cũ (MD thô). `mode: obsidian` = bật các tính năng dưới. Vault có thể là `.hatch/kb/` (mặc định) hoặc một vault ngoài (chia sẻ với người).

## Tương thích Obsidian

- **Wikilinks**: `related:` và liên kết giữa note render thành `[[ADR-007 CSV streaming]]`, hỗ trợ `[[note#heading]]` và alias `[[note|tên hiển thị]]`. Hatch resolve được wikilink ↔ file.
- **Tags**: dùng cả frontmatter `tags:` lẫn `#tag` trong thân — khớp Obsidian.
- **Frontmatter**: giữ YAML hiện có (id/type/title/tags/related/...), thêm `aliases:` (Obsidian) khi cần.
- **MOC (Map of Content)**: `kb/index.md` trở thành MOC bằng wikilink, nhóm theo type/tag — Obsidian hiển thị graph + outline.
- **Backlinks**: `hatch kb` tự tính backlink (note nào trỏ tới đây) để bổ trợ index; Obsidian cũng tự có pane backlink.
- **Dataview (tuỳ chọn)**: sinh frontmatter chuẩn để user viết query Dataview (bảng ADR theo status, learnings theo tag) sống động trong app.

## CLI

```bash
hatch kb add --type decision --title "CSV streaming" --tags export --link T-123,ADR-003
hatch kb link <from> <to>          # tạo wikilink hai chiều
hatch kb backlinks <note>          # ai trỏ tới note này
hatch kb graph [--tag export]      # in graph text (note ↔ liên kết) — recall theo đồ thị
hatch kb open <note>               # mở trong Obsidian qua obsidian:// URI
hatch kb index                     # build MOC index.md (wikilink, nhóm type/tag)
hatch kb sync                      # đối chiếu vault ↔ .meta.json, sửa link gãy
```

## Recall theo đồ thị (mạnh cho token)

Ngoài `hatch search` (token match), Obsidian-mode cho **graph-aware recall**: "nạp ADR-007 **và mọi note nó link tới**" — agent lấy đúng cụm tri thức liên quan theo wikilink/backlink thay vì đọc cả vault. Đây là L2 on-demand nâng cấp.

## Caveat trung thực
- Obsidian **không có CLI/đầu mối tự động chính thức**; tích hợp dựa trên (a) file trực tiếp, (b) URI mở app, (c) plugin Local REST API (chỉ khi app chạy). Hatch không phụ thuộc app đang mở — vault-as-files là chính.
- `mode: native` luôn là fallback an toàn; Obsidian là lớp tăng cường, không phải phụ thuộc cứng.
- Sync xung đột nếu vault vừa do người sửa trong app vừa do agent ghi — dựa git như mọi artifact khác.
