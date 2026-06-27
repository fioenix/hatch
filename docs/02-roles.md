# 02 — Roles

## Tách bạch: Vai ≠ Agent

Nguyên tắc cốt lõi, mượn từ squad người: **vai (role)** là một bộ trách nhiệm + ranh giới + context; **agent** là một thực thể thực thi. Một người có thể đảm nhiều vai; một vai có thể do nhiều người luân phiên. Hatch giữ đúng tách bạch này:

- **Role** định nghĩa: làm gì, được phép làm gì, nạp context nào (L1).
- **Agent** định nghĩa: năng lực, lệnh CLI, surface đọc instruction.
- **Registry** là bảng phân công nối hai cái lại.

Đổi agent cho một vai = sửa 1 dòng registry + compile lại. Không phải viết lại context.

> **Vai là cấu hình per-project, không fix cứng.** `registry.yaml` sống trong `.hatch/` của *từng project*. Cùng một agent (vd Codex) có thể là `Implementer` ở project A, `Tester` ở project B, hoặc kiêm cả hai ở project C — do **người dùng quyết định ở từng project**. Bộ vai chuẩn + bảng map dưới đây chỉ là **template khởi đầu** để khỏi bắt đầu từ số 0; user sửa thoải mái.

## Bộ vai chuẩn (template khởi đầu)

Lấy theo một squad sản phẩm điển hình — copy rồi sửa cho project của bạn:

| Vai | Trách nhiệm | Ranh giới (KHÔNG được) | Context L1 nạp |
|---|---|---|---|
| **Conductor** | Lập kế hoạch, bẻ epic→ticket, gán vai/agent, gỡ block, chạy ceremonies | Tự viết code production lớn; tự merge | charter + board tổng + roadmap |
| **Architect** | Spec kỹ thuật, design, ADR, đặt ràng buộc | Bỏ qua review; quyết business | tech/ + product/ |
| **Implementer** | Viết code theo ticket + DoD | Đổi scope ticket; sửa file ngoài ticket | role + ticket + context_refs |
| **Reviewer** | Review diff, gác DoD, phê duyệt | Tự sửa rồi tự duyệt (xung đột lợi ích) | tech/ conventions + ticket + DoD |
| **Tester** | Viết/chạy test, báo cáo, dựng repro | Sửa code production để test pass | tech/test + ticket |
| **Tech Writer** | Docs, changelog, runbook | Đổi hành vi code | product/ + ticket |

Người dùng thêm/bớt/đổi tên vai tùy project (vd: `SecurityReviewer`, `DataEngineer`, `PromptEngineer`). Một project nhỏ có thể chỉ cần 2 vai (Implementer + Reviewer); một project lớn có thể có 8 vai.

## Map năng lực → agent (template gợi ý, override per-project)

Phân vai theo điểm mạnh quan sát được của từng agent. Đây **chỉ là gợi ý xuất phát**; người dùng tự đặt lại trong `registry.yaml` của mỗi project.

| Agent | Vai phù hợp | Lý do |
|---|---|---|
| **Claude Code** | Conductor, Architect, Reviewer | Reasoning dài, giữ context lớn, phán đoán trade-off, planning |
| **Kiro** | Architect (spec), Implementer | Quy trình spec-driven PRD→design→tasks bài bản |
| **Codex** | Implementer, Tester | Vòng lặp code-test nhanh, autonomous |
| **Antigravity CLI** | Implementer, Tech Writer, Utility | Linh hoạt, gánh việc nền |

### Vì sao CC làm Conductor

Trong Hybrid, Conductor là vai cần bức tranh tổng thể + phán đoán ưu tiên — đúng điểm mạnh CC, và CC là agent chính của người dùng nên giữ vai trung tâm hợp lý. Conductor **không độc quyền code**; nó điều phối rồi giao cho workers.

## Một agent giữ nhiều vai

Hoàn toàn được, như một dev kiêm reviewer cho ticket người khác. Ràng buộc đạo đức (mượn từ người): **không tự review chính mình**. Registry enforce: nếu `assignee` của ticket = agent X ở vai Implementer, thì vai Reviewer của ticket đó phải là agent ≠ X (hoặc human).

## Binding trong registry (xem [spec](../spec/registry.schema.md))

```yaml
agents:
  claude-code:
    cli: "claude"
    surface: ["CLAUDE.md", ".claude/"]
    roles: [conductor, architect, reviewer]
  kiro:
    cli: "kiro"
    surface: [".kiro/steering/"]
    roles: [architect, implementer]
  codex:
    cli: "codex"
    surface: ["AGENTS.md"]
    roles: [implementer, tester]
  antigravity:
    cli: "ag"
    surface: ["<config>"]
    roles: [implementer, tech-writer]

policy:
  no-self-review: true          # implementer ≠ reviewer trên cùng ticket
  require-human-gate: [deploy, secret, external-comms]
```
