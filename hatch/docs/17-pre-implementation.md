# 17 — Chốt trước khi implement toàn bộ backlog

> Danh sách quyết định cần xử lý trước/trong lúc code phần còn lại. Mỗi mục có **đề xuất mặc định**; gắn 🔴 = cần user quyết, 🟡 = mặc định hợp lý nhưng nên xác nhận, 🟢 = tự quyết được.

## Kỹ thuật cốt lõi

1. 🟡 **Kiểm thử với agent thật.** Remote env không có `claude/codex/gemini/kiro-cli` → mới verify ở mức dry-run + unit test. *Đề xuất:* build một **mock agent CLI** (đọc prompt, in output giả + cost giả) để test trọn nhánh execute/relay/pair/convene end-to-end; chạy thật với agent xịn để CI/local sau.
2. 🔴 **Cost capture.** Mỗi agent trả usage khác nhau (Claude `total_cost_usd`, Codex/Gemini token usage). Giá thay đổi. *Đề xuất:* parse tokens từ JSON output mỗi adapter, lưu **tokens thô** + nhân với **bảng rate cấu hình** trong registry (không hardcode giá). Cần xác nhận: dùng giá tự khai hay cố đọc cost provider trả về.
3. 🟡 **Concurrency / song song thật.** `watch` hiện tuần tự. Chạy nhiều agent song song cần: worktree mỗi ticket (đã có helper), khóa qua git push, xử lý xung đột merge, giới hạn parallelism. *Đề xuất:* goroutine pool theo WIP toàn cục + worktree-per-ticket; claim vẫn "push thắng = lock thắng".
4. 🟡 **Bus parser.** Body chứa `## ` làm parser tách nhầm message. *Đề xuất:* escape `## ` đầu dòng khi ghi (hoặc dùng delimiter ít đụng hơn) — sửa trước khi dùng convene/pair nặng.
5. 🟢 **Nén ledger/bus.** Append-only sẽ phình. *Đề xuất:* retro nén ledger theo chu kỳ (đã có khái niệm); bus archive thread cũ sang `bus/archive/`. Implement khi cần.

## Bảo mật / vận hành

6. 🔴 **Secrets cho agent headless.** `ANTHROPIC_API_KEY`, `KIRO_API_KEY`… *Đề xuất:* chỉ qua env/secret manager, **không bao giờ trong repo**; registry chỉ tham chiếu tên biến. (Khớp chính sách "không hardcode credentials".)
7. 🔴 **Redaction.** Output agent capture vào ledger/bus có thể lộ dữ liệu nhạy cảm. *Đề xuất:* lớp redact (mask token/PII pattern) trước khi ghi; cấu hình pattern trong charter. Cần quyết mức độ.
8. 🟡 **Sandbox mặc định.** Mỗi adapter map sang quyền (codex `-s`, claude `--permission-mode`). *Đề xuất:* mặc định **bảo thủ** (workspace-write / acceptEdits), nâng quyền phải khai trong registry.

## Phạm vi / sản phẩm

9. 🟡 **Obsidian vault location.** Trong repo (`.hatch/kb`) hay vault ngoài? *Đề xuất:* mặc định trong repo (versioned, đơn giản); cho trỏ vault ngoài qua config.
10. 🟢 **Ranh giới nơi lưu tài liệu.** ADR→`kb/`, PRD→`context/product`, design→spec/`context/tech` (đã định ở [16](16-document-templates.md)).
11. 🟡 **Thứ tự implement.** *Đề xuất* (gộp [14](14-org-and-cadence.md) §thứ-tự + 2 mục mới):
    1. Hardening nền: mock agent, bus parser fix, secrets/redaction.
    2. Cost capture + `budget`/`cost` + auto-pause (nối presence).
    3. `workload` + `perf` (đọc ledger).
    4. Doc templates (`doc new/lint`) + Obsidian KB mode.
    5. Org-chart/DoA + external deps.
    6. `tick` (cadence) + `report` + parallel `watch`.
12. 🟢 **Versioning/migration.** registry/workflow có `version`; thêm migration nhẹ khi đổi schema.

## Quyết định đã chốt (2026-06-14)
- **Mock agent: CÓ.** Dựng `hatch-mock` + adapter `kind: mock` để test end-to-end execute/relay/pair/convene/cost trong remote.
- **Obsidian vault: CẢ HAI.** Hỗ trợ qua config; **mặc định in-repo** (`.hatch/kb`), cho trỏ vault ngoài.
- **Cost/secrets: TỐI GIẢN.** Chỉ **track** cost/tokens (không hard-cap, không auto-pause, không redaction mặc định). *Ngoại lệ bất biến:* secrets vẫn **chỉ qua env**, không bao giờ trong repo (quy tắc cứng, độc lập với mức "tối giản").

Mặt thiết kế đã đủ phủ một human squad (vận hành + giao tiếp + quản trị). Bắt đầu implement theo thứ tự #11.
