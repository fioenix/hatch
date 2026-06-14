# 07 — Orchestrator (`hatch` CLI)

Phase 3 đích đến: một CLI `hatch` đóng vai **Engineering Manager / Scrum Master** — không tự viết code, mà *compile context, lập kế hoạch, spawn đúng agent cho đúng ticket, gác gate, và dựng dashboard*. Phase 1–2 nhiều việc dưới đây làm thủ công; orchestrator chỉ tự động hóa lại.

> Đúng nghĩa cái tên: `hatch run <ticket>` là *nở* một agent ra khỏi ổ — spawn đúng agent, đặt vào worktree riêng với context vừa đủ, để nó đi làm rồi báo cáo lại qua ledger.

## Lệnh

| Lệnh | Vai trò người tương đương | Làm gì |
|---|---|---|
| `hatch init` | Lập đội + dựng quy trình | Scaffold `.hatch/` (charter, roles mẫu, protocol, board, registry) |
| `hatch compile [--check]` | Phát handbook onboarding | SSOT → file native từng agent; `--check` chặn nếu stale |
| `hatch plan` | Sprint planning | Conductor (CC) bẻ epic→ticket, gán vai/agent, xếp ưu tiên/dependency |
| `hatch run <ticket>` | Giao việc cho dev | Spawn CLI của agent được gán, với prompt + context scoped (L0+L1+L2), trong worktree riêng |
| `hatch gate <ticket>` | Review + CI | Chạy DoD/test/lint; cập nhật trạng thái |
| `hatch status` | Bảng sprint | Dashboard: lane, ai làm gì, blocker, stale |
| `hatch standup` | Daily standup | Sinh ledger digest từ entry mới |
| `hatch sync` | Đồng bộ thực tế | Đối chiếu board ↔ git ↔ PR, sửa lệch |

## `hatch run` — cơ chế spawn (cốt lõi Phase 3)

Đây là khác biệt giữa "convention" (Phase 1) và "orchestrator thật" (Phase 3).

```
act run T-123
   │
   1. đọc ticket T-123 → biết assignee agent, role, context_refs
   2. compile prompt scoped:
        L0 charter  +  L1 role(assignee)  +  L2 (ticket body + context_refs)
        +  protocol cần thiết (claim, DoD, branching)
   3. tạo git worktree hatch/T-123-<slug>  (cô lập, chạy song song an toàn)
   4. spawn CLI của agent đó (vd: `codex`, `kiro`, `claude`, `ag`)
        với prompt ở trên, cwd = worktree
   5. stream stdout/stderr → ledger entry (audit)
   6. khi agent xong → orchestrator:
        - chạy gate tự động (test/lint)
        - git mv ticket sang review/
        - mở PR draft
        - ghi ledger
```

### Adapter spawn theo agent

Mỗi agent có cách invoke khác nhau (cờ, cách nhận prompt, chế độ non-interactive). Hatch có một **spawn adapter** per agent (song song với compile adapter ở [compiler](04-context-compiler.md)):

```yaml
# trong registry.yaml, mỗi agent
codex:
  spawn:
    cmd: "codex"
    args: ["--non-interactive", "--prompt-file", "{prompt}"]
    cwd: "{worktree}"
    capture: stdout
```

Thêm agent mới = thêm compile adapter + spawn adapter + dòng registry. Không đụng lõi.

## Cô lập bằng git worktree

Chạy nhiều agent song song an toàn nhờ mỗi ticket một worktree riêng (cùng repo, khác thư mục làm việc, khác branch). Không tranh working tree, không tranh index. Đây là tương đương "mỗi dev một máy/branch".

## Conductor là agent, không phải code

`hatch plan` không nhồi logic planning vào CLI — nó **spawn CC ở vai Conductor** với context board + roadmap, và CC trả về tập ticket. Orchestrator chỉ là khung process + I/O + gate; *phán đoán* vẫn do agent. Giữ đúng triết lý: tool điều phối, agent suy nghĩ.

## Điều orchestrator KHÔNG làm

- Không tự merge / deploy / chạm secret — đó là human gate ([governance](06-governance.md)).
- Không tự sửa file output compiled — chỉ regenerate từ SSOT.
- Không quyết định business/scope — chỉ thực thi kế hoạch Conductor + ranh giới registry.

## Lựa chọn kỹ thuật (đề xuất, chốt khi implement)

- **Ngôn ngữ:** một CLI gọn (Go hoặc Node/TS) để dễ phân phối 1 binary; hoặc Python nếu ưu tiên tốc độ làm.
- **Trạng thái:** thuần file + git, không DB (xem [architecture](01-architecture.md)).
- **Config:** `registry.yaml` + `.hatch/`.
- **Quan sát:** `hatch status` đọc trực tiếp board/ledger; tùy chọn TUI.

Các lựa chọn này để mở; xem [roadmap](08-roadmap.md) Phase 3 để chốt.
