# Spec — `workflow.yaml`

Định nghĩa quy trình làm việc của một project: lanes, transitions, gates, ceremonies. **Đây là template — user sửa tự do per-project.** Board, protocol, và orchestrator đều đọc từ file này; đổi quy trình = sửa file, không sửa code.

## Schema

```yaml
version: 1
template: scrum            # scrum | kanban | spec-first | lite | custom

# Các trạng thái = thư mục trong board/
lanes:
  - id: backlog
  - id: in-progress
    wip-limit: 2           # tùy chọn (Kanban)
  - id: review
  - id: done
  - id: blocked
    side: true             # lane phụ, không nằm trên luồng chính

# Chuyển trạng thái hợp lệ + ai được làm + điều kiện
transitions:
  - from: backlog
    to: in-progress
    by: [implementer, tester]      # vai được phép (khớp registry)
    action: claim                  # ghi ledger action gì
  - from: in-progress
    to: review
    by: [implementer]
    gates: [tests-pass, lint-clean, handoff-note]
  - from: review
    to: done
    by: [reviewer]
    gates: [dod-met, no-self-review, human-merge]
  - from: review
    to: in-progress
    by: [reviewer]
    action: changes-requested
  - from: "*"
    to: blocked
    by: ["*"]
    action: block

# Cổng — điều kiện pass khi qua transition
gates:
  tests-pass:    { type: command, run: "make test" }
  lint-clean:    { type: command, run: "make lint" }
  dod-met:       { type: checklist, ref: protocol/definition-of-done.md }
  handoff-note:  { type: required-field, field: handoff }      # bắt buộc có handoff note
  no-self-review:{ type: policy, ref: registry.policy.no-self-review }
  human-merge:   { type: human-gate }                          # dừng chờ người

# Sự kiện định kỳ + ai chủ trì
ceremonies:
  planning:        { by: conductor, trigger: manual }          # sprint planning
  standup-digest:  { by: conductor, trigger: "daily" }         # ledger digest
  retro:           { by: conductor, trigger: "end-of-sprint", actions: [compact-ledger, promote-kb] }

# Cấu hình spec-driven (nếu bật)
spec:
  required-for: [epic]          # loại nào bắt buộc qua PRD→Design→Tasks
  artifacts: [prd, design, tasks]
  gates:
    prd:    human-gate
    design: review
```

## Quy tắc

- **`lanes`** sinh ra thư mục `board/<lane>/`. Đổi lane = đổi cấu trúc board.
- **`transitions`** là luật duy nhất quyết định chuyển trạng thái hợp lệ. `by` phải khớp vai trong [registry](registry.schema.md). `from: "*"` = áp cho mọi lane.
- **`gates`** chạy khi một transition khai báo nó; fail → transition bị chặn. `human-gate` luôn dừng chờ người ([governance](../docs/06-governance.md)).
- **`ceremonies`** map sang cơ chế async ([workflow](../docs/05-workflow.md#ceremonies--cơ-chế-async)); `retro` có thể kéo theo `promote-kb` (đề bạt KB→SSOT) và `compact-ledger`.
- Bỏ trống `spec` = tắt spec-driven (đi thẳng backlog).

## Template ship sẵn

| Template | Lanes | Đặc điểm |
|---|---|---|
| `scrum` | backlog→in-progress→review→done | sprint + retro định kỳ (mặc định) |
| `kanban` | + wip-limit | pull liên tục, không sprint |
| `spec-first` | + cổng prd/design | mọi epic qua PRD→Design→Tasks |
| `lite` | todo→doing→done | tối giản cho project nhỏ/cá nhân |
| `dual-track` | ideas→discovery→ready→in-progress→review→done | dual-track agile: discovery song song delivery |
| `shape-up` | pitch→bet→building→review→done | Shape Up: cược scope đã shaped, appetite cố định |
| `stage-gate` | requirements→design→build→test→release→done | PDLC phân pha, sign-off mỗi cổng |
| `incident` | detected→triage→mitigating→resolved→postmortem | ứng cứu sự cố; ghép `hatch oncall` + escalation |

User chọn một template rồi sửa, hoặc đặt `template: custom` và tự khai báo toàn bộ.

## Validate (pre-commit / orchestrator)

- Mọi `lane` tham chiếu trong `transitions` phải tồn tại.
- `by` của transition phải là vai có trong registry.
- Mọi `gates[]` được transition dùng phải định nghĩa trong khối `gates`.
- Đồ thị lanes phải tới được `done` (không có lane chết).
