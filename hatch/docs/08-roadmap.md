# 08 — Roadmap

Đích cuối là full orchestrator, nhưng triển khai theo 3 phase để **chạy được giá trị ngay từ Phase 1** mà không cần viết code.

## Phase 1 — Convention + Docs (chạy được không cần code)

Mọi thứ là file + git + quy ước. Agent tuân theo protocol thủ công; con người (hoặc CC ở vai Conductor) chạy các bước bằng tay.

**Giao được:**
- Bộ docs này (đã có).
- `hatch init` dạng **template thư mục** `.hatch/` để copy vào repo (charter mẫu, roles mẫu, protocol, board rỗng, registry mẫu, `kb/` rỗng + `index.md`, `workflow.yaml` chọn template scrum/kanban/spec-first/lite).
- Compile **thủ công**: hướng dẫn + checklist để CC tự sinh `CLAUDE.md`/`AGENTS.md`/`.kiro/steering/` từ SSOT (tái dùng skill overclaud).
- Một **skill/agent overclaud** mở rộng: "thiết lập Hatch cho repo này" — hỏi vai, agent, rồi dựng `.hatch/`.

**Tiêu chí xong:** một repo thật chạy được vòng `plan → claim → code → review → done` hoàn toàn bằng convention, có ledger.

## Phase 2 — CLI hỗ trợ (tự động hóa phần cơ học)

Thêm CLI `hatch` cho các thao tác xác định, **chưa spawn agent**:

**Giao được:**
- `hatch init` (scaffold thật).
- `hatch compile [--check]` — SSOT → output per-agent + manifest stale detection.
- `hatch status` / `hatch standup` — đọc board/ledger, dashboard + digest.
- `hatch gate` — chạy test/lint/DoD checklist (đọc gates từ `workflow.yaml`).
- `hatch kb query <tag>` / `hatch kb add` — tra cứu + ghi Knowledge Base, cập nhật `index.md`.
- `hatch sync` — đối chiếu board ↔ git ↔ PR.
- git pre-commit hook: chặn output stale, validate frontmatter ticket + `workflow.yaml`/`registry.yaml`.

**Tiêu chí xong:** không còn thao tác cơ học thủ công; agent vẫn được người/Conductor gọi tay, nhưng compile/status/gate đã tự động.

## Phase 3 — Full Orchestrator (spawn & drive agent)

Orchestrator tự spawn đúng agent cho đúng ticket (xem [orchestrator](07-orchestrator.md)).

**Giao được:**
- `hatch run <ticket>` — spawn CLI agent với context scoped, trong worktree, capture → ledger.
- `hatch plan` — spawn CC (Conductor) để bẻ epic→ticket.
- Spawn adapter per agent trong registry.
- Worktree isolation cho chạy song song.
- Tùy chọn: `hatch watch` chạy vòng lặp pull-ticket tự động cho workers (có WIP limit).

**Tiêu chí xong:** từ một backlog, `hatch` tự lập kế hoạch, phân việc, chạy nhiều agent song song, dừng đúng ở mọi human gate.

## Phụ thuộc giữa phase

```
Phase 1 (convention)  ──►  Phase 2 (CLI cơ học)  ──►  Phase 3 (spawn)
   SSOT + protocol           compile + status           run + plan
   board + ledger            gate + sync                worktree + adapters
```

Mỗi phase đứng vững một mình. Phase 1 đã có giá trị (hết drift, có quy trình, có audit). Phase 2 bỏ việc tay. Phase 3 bỏ việc gọi agent tay.

## Câu hỏi cần chốt trước khi code (Phase 2+)

1. Ngôn ngữ CLI: Go (1 binary, dễ phân phối) vs Node/TS (gần hệ sinh thái agent) vs Python (làm nhanh)?
2. Compile adapter: viết riêng từng agent, hay một format trung gian + renderer?
3. Mức độ tự động của `hatch watch` ở Phase 3 — pull tự động tới đâu trước khi cần người?
4. Tích hợp CI thật (GitHub Actions) cho gate, hay chạy local?

Những câu này không chặn Phase 1. Chốt khi tới Phase 2.

## Việc kế tiếp ngay

1. Review & chốt bộ docs này.
2. Dựng template `.hatch/` (Phase 1) như một thư mục trong repo overclaud.
3. Mở rộng skill overclaud để bootstrap Hatch cho một repo.
