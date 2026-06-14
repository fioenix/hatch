# 03 — Coordination Protocol

Đây là trái tim của Hatch: làm sao nhiều agent **không chung bộ nhớ, mỗi con một process** phối hợp mà không giẫm chân. Câu trả lời mượn nguyên từ squad người: **phối hợp qua artifact chung, bất đồng bộ, có quy ước rõ.**

## Mô hình Hybrid

```
   ┌──────────── PLAN (đồng bộ, định kỳ) ────────────┐
   │  Conductor: epic → tickets, gán vai/agent,      │
   │  set ưu tiên + dependency, đẩy vào backlog/     │
   └────────────────────┬────────────────────────────┘
                        │
   ┌────────────────────▼──────── EXECUTE (bất đồng bộ) ──────────┐
   │  Workers tự CLAIM ticket trong lane vai mình →               │
   │  làm trên branch riêng → ledger → đẩy sang review/           │
   │  Reviewer GATE → done/ ;  bất kỳ ai gặp block → mở blocker   │
   └──────────────────────────────────────────────────────────────┘
```

- **Pha PLAN** giống sprint planning: tập trung, do Conductor chủ trì, sinh ra ticket có chủ.
- **Pha EXECUTE** giống ngày thường: workers tự kéo việc, async, phối hợp qua board + ledger.

Không có lời gọi trực tiếp agent→agent. Mọi "giao tiếp" là **ghi vào file**, "lắng nghe" là **đọc file**. Đúng như dev không ghi đè não đồng nghiệp mà comment lên Jira/PR.

## Board như một state machine

Trạng thái ticket = thư mục chứa nó:

```
backlog/  →  in-progress/  →  review/  →  done/
                  │                ↑
                  └──── blocked/ ──┘   (lane phụ; Conductor gỡ)
```

Chuyển trạng thái = `git mv` file ticket sang thư mục mới + cập nhật frontmatter + ghi ledger. Một chuyển = một commit. Diff của commit chính là biên bản.

## Claim / Lock — chống hai agent cùng một việc

git là cơ chế lock tự nhiên. Quy ước:

1. **Claim:** worker muốn nhận ticket → `git mv backlog/T-123.md in-progress/` + set frontmatter `assignee: <agent>`, `claim: {agent, ts, branch}` → commit → **push ngay**.
2. **Lock thắng = push thắng.** Nếu hai agent claim cùng lúc, người push trước thắng; người thua bị reject khi push (non-fast-forward), pull về thấy ticket đã có chủ → bỏ, claim cái khác.
3. **Branch-per-ticket:** mỗi ticket làm trên `hatch/T-123-<slug>`. Cô lập file, tránh đụng working tree. (Phase 3: dùng git worktree để chạy song song thật.)
4. **TTL claim:** nếu một claim quá `stale_after` (vd 2h) không có ledger update → Conductor được quyền thu hồi (giống reassign khi đồng nghiệp mất tích).

Chi tiết: [protocol/claim-lock](#) — sẽ sinh vào `.hatch/protocol/claim-lock.md` khi init.

## Hand-off giữa các vai

Hand-off = ticket đổi `role` + đổi thư mục, kèm một **handoff note** trong ledger nêu: đã làm gì, còn gì, cần gì ở vai sau.

Ví dụ chuỗi một ticket feature:
```
Architect (design xong) ──► Implementer (code) ──► Reviewer (gate) ──► done
     ↑ handoff note: spec ở đâu, ràng buộc gì
                          ↑ handoff note: branch nào, test nào chạy
```

Handoff note bắt buộc khi đổi assignee — đây là "context tối thiểu để người sau không phải đọc lại toàn bộ", chính là cơ chế tiết kiệm token ở tầng phối hợp.

## Dependency & thứ tự

Ticket khai báo `depends_on: [T-100, T-101]`. Worker chỉ được claim khi mọi dependency đã ở `done/`. Conductor dùng đồ thị dependency để xếp ưu tiên ở pha PLAN. Vòng lặp dependency = lỗi, Conductor phát hiện lúc plan.

## Blocker

Bất kỳ vai nào gặp chặn → set `status: blocked`, `git mv` sang `blocked/`, ghi ledger nêu lý do + cần gì. Conductor quét `blocked/` mỗi lần plan/standup, gỡ hoặc leo thang cho human.

## Definition of Done (DoD)

Mỗi ticket kế thừa DoD mặc định + DoD riêng. Reviewer gác đúng checklist này. Mặc định:

- [ ] Code khớp scope ticket, không đụng file ngoài `context_refs` không khai báo
- [ ] Test liên quan pass (Tester/CI xác nhận)
- [ ] Lint/format sạch
- [ ] Diff được review bởi agent ≠ implementer (no-self-review)
- [ ] Ledger có entry hoàn thành + handoff note
- [ ] Branch có PR (human gate cho merge)

Xem thêm [governance](06-governance.md) về gate trước merge.

## Vì sao async-board, không phải agent gọi agent

| Tiêu chí | Async board (chọn) | Gọi trực tiếp |
|---|---|---|
| Heterogeneous CLI | Không cần API chung | Cần protocol liên-CLI (không tồn tại) |
| Audit | Mọi thứ là commit | Khó truy vết |
| Lỗi cô lập | Một agent chết không kéo cả hệ | Caller treo theo |
| Human đọc xen được | Có (đọc/sửa file) | Khó |
| Token | Chỉ nạp ticket cần | Dễ nhồi cả hội thoại |
