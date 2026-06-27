# 04 — Context & Compiler

Hai mục tiêu overclaud nâng lên multi-agent: **hết drift** (1 nguồn → N output) và **tiết kiệm token** (mỗi agent nạp tối thiểu).

## SSOT vs Knowledge Base — hai loại tri thức

Hatch tách hai loại tri thức vì chúng có vòng đời và quyền ghi khác nhau:

| | **SSOT** (`context/`, `charter.md`, `roles/`) | **Knowledge Base** (`kb/`) |
|---|---|---|
| Là gì | *Config/chuẩn* để định hình agent | *Tri thức tích lũy* khi làm việc |
| Hướng | Đầu vào → **compile** vào prompt | Vào **và** ra; tra cứu on-demand |
| Ai ghi | Human / Architect (cẩn trọng, ít đổi) | Mọi agent (ghi liên tục khi học được) |
| Bị compile? | **Có** → `CLAUDE.md`/`AGENTS.md`… | **Không** — đọc động qua index/truy vấn |

Doc này nói về SSOT + compile. Chi tiết KB ở [09-knowledge-base](09-knowledge-base.md).

## Single Source of Truth (SSOT)

Tất cả tri thức canonical (config) sống một nơi: `.hatch/context/` + `.hatch/charter.md` + `.hatch/roles/`. **Không bao giờ** viết tay vào `CLAUDE.md`/`AGENTS.md`/`.kiro/steering/` — chúng là output sinh ra.

```
.hatch/charter.md          → mission, value, ràng buộc tối cao   (L0)
.hatch/roles/<role>.md     → trách nhiệm + ranh giới + cách làm   (L1)
.hatch/context/
  ├── product/           → PRD, domain, business rules
  ├── tech/              → stack, conventions, kiến trúc, ADR
  └── shared.md          → điều mọi vai cần biết
```

## Ba tầng token (L0 / L1 / L2)

Mô phỏng cách một nhân viên thật mang theo thông tin: ai cũng thuộc *mission công ty* (L0), nắm *JD của mình* (L1), và chỉ mở *hồ sơ việc đang làm* khi cần (L2).

| Tầng | Nội dung | Ai nạp | Khi nào |
|---|---|---|---|
| **L0** Mission | charter — rất ngắn | Mọi agent, mọi message | Luôn (nằm trong file compiled) |
| **L1** Role | role file của (các) vai agent giữ | Agent theo vai | Luôn (trong file compiled) |
| **L2** Task | ticket active + `context_refs` + **mục KB liên quan** | Agent đang làm ticket | On-demand, chỉ khi claim |

**Nguyên tắc vàng:** file compiled (luôn nạp mỗi message) chỉ chứa L0 + L1 + **con trỏ** tới L2. L2 (gồm cả tra cứu KB) được agent đọc *khi* nhận ticket, không nhồi sẵn. Đây là khác biệt token lớn nhất so với "một CLAUDE.md chứa tất cả". KB cũng giúp tiết kiệm token theo cách khác: agent **tra cứu quyết định/bài học có sẵn** thay vì suy diễn lại từ đầu.

### Ước lượng tiết kiệm

Giả sử context đầy đủ ~ 8.000 token. Naïve: mỗi agent nạp tất cả mỗi message.

| Cách | Token/message/agent |
|---|---|
| Naïve (nạp tất cả) | ~8.000 |
| Hatch (L0 ~300 + L1 ~700 + con trỏ) | ~1.000; L2 chỉ nạp khi cần |

Với 4 agent × nhiều message/ngày, chênh lệch tích lũy rất lớn (xem bảng lãng phí token trong [overclaud README](../../README.md)).

## Compiler: SSOT → per-agent

```
                    ┌──────────────┐
  charter (L0) ─────┤              │
  roles  (L1) ──────┤   COMPILER   ├──► CLAUDE.md            (backend: overclaud)
  registry binding ─┤              ├──► AGENTS.md            (backend: codex)
  context (con trỏ)─┤              ├──► .kiro/steering/*.md  (backend: kiro)
                    └──────┬───────┘──► <antigravity config> (backend: antigravity)
                           ▼
                    compiled/.manifest.json  (hash nguồn để phát hiện stale)
```

### Quy trình compile cho mỗi agent

1. Đọc registry → biết agent giữ vai gì, surface nào.
2. Ghép `charter` (L0) + role file của các vai đó (L1) + danh sách con trỏ context (đường dẫn tới `context/`, không nhúng nội dung).
3. Áp **adapter theo agent** (định dạng/cú pháp native):
   - Claude Code → cú pháp `CLAUDE.md`, có thể tách `.claude/rules/` (tái dùng templates overclaud).
   - Codex → `AGENTS.md`.
   - Kiro → nhiều file `.kiro/steering/` (Kiro thích tách mảnh + spec).
   - Antigravity → theo config của nó.
4. Áp **token pass** (nguyên tắc overclaud: bỏ thừa, mỗi từ đáng giá).
5. Ghi output + cập nhật `manifest` (hash nguồn).

### Adapter là điểm mở rộng

Thêm một agent mới = viết một adapter (SSOT → định dạng của nó) + thêm dòng registry. Phần SSOT không đổi. Đây là cách Hatch giữ "viết một lần, chạy mọi agent".

## Stale detection

`manifest.json` lưu hash của SSOT lúc compile gần nhất. Nếu SSOT đổi mà chưa compile lại → file output là **stale**. Cảnh báo:
- Phase 1: lệnh/checklist thủ công + git pre-commit hook gợi ý.
- Phase 2/3: `hatch compile --check` chặn ở CI / pre-commit; orchestrator tự compile trước khi `run`.

## Quy tắc bất biến

1. Output (`CLAUDE.md`, `AGENTS.md`, `.kiro/steering/`…) **không sửa tay** — sửa SSOT rồi compile.
2. Mỗi mẩu tri thức sống **đúng một chỗ** trong SSOT (DRY). Nhiều vai cần → để ở `shared.md` hoặc `context/`, role file chỉ trỏ tới.
3. File compiled = L0 + L1 + con trỏ; **không nhúng L2**.
