# Spec — `registry.yaml`

Bảng phân công của đội: roster agent, năng lực, binding vai, policy. Nguồn để compiler và orchestrator biết "ai là ai, làm được gì, được gọi thế nào".

## Schema

```yaml
version: 1

# Định nghĩa vai (hoặc trỏ tới .hatch/roles/*.md)
roles:
  conductor:    { file: roles/conductor.md }
  architect:    { file: roles/architect.md }
  implementer:  { file: roles/implementer.md }
  reviewer:     { file: roles/reviewer.md }
  tester:       { file: roles/tester.md }

# Roster agent
agents:
  claude-code:
    cli: "claude"                       # binary để invoke
    surface: ["CLAUDE.md", ".claude/"]  # nơi compiler ghi output
    roles: [conductor, architect, reviewer]
    compile:
      adapter: claude                   # adapter SSOT → định dạng native
    spawn:                              # (Phase 3) cách orchestrator spawn
      cmd: "claude"
      args: ["-p", "{prompt}"]
      cwd: "{worktree}"
      capture: stdout

  kiro:
    cli: "kiro"
    surface: [".kiro/steering/"]
    roles: [architect, implementer]
    compile: { adapter: kiro }
    spawn:   { cmd: "kiro", args: ["run", "--task", "{prompt}"], cwd: "{worktree}", capture: stdout }

  codex:
    cli: "codex"
    surface: ["AGENTS.md"]
    roles: [implementer, tester]
    compile: { adapter: codex }
    spawn:   { cmd: "codex", args: ["--non-interactive", "--prompt-file", "{prompt}"], cwd: "{worktree}", capture: stdout }

  antigravity:
    cli: "ag"
    surface: ["<antigravity-config>"]
    roles: [implementer, tech-writer]
    compile: { adapter: antigravity }
    spawn:   { cmd: "ag", args: ["task", "{prompt}"], cwd: "{worktree}", capture: stdout }

# Quy ước phối hợp / governance
policy:
  no-self-review: true                              # implementer ≠ reviewer cùng ticket
  require-human-gate: [merge, deploy, secret, external-comms, destructive-data]
  branch-pattern: "hatch/{ticket}-{slug}"
  claim-stale-after: "2h"                           # quá hạn → Conductor thu hồi
  wip-limit:                                        # giới hạn ticket đồng thời / vai (Kanban)
    implementer: 2
    reviewer: 3

# Biến thể quy trình (xem workflow.md)
workflow:
  mode: scrum            # scrum | kanban
  spec-required-for: epic   # epic bắt buộc qua PRD→Design→Tasks
```

## Quy tắc

- `roles` của mỗi agent quyết định **L1** nào được ghép vào file compiled của nó.
- `surface` quyết định compiler ghi output ở đâu.
- `compile.adapter` và `spawn` là hai điểm mở rộng độc lập; thêm agent mới chỉ cần khai báo ở đây + viết adapter tương ứng.
- `{prompt}` / `{worktree}` / `{ticket}` / `{slug}` là biến orchestrator nội suy lúc `hatch run`.
- `policy` được enforce: bởi convention (Phase 1), pre-commit hook (Phase 2), orchestrator (Phase 3).

## Validate

- Mọi vai trong `agents[].roles` phải tồn tại trong `roles`.
- Mỗi vai chuẩn nên có ≥ 1 agent đảm nhiệm (cảnh báo nếu vai không ai giữ).
- `no-self-review: true` → cần ≥ 2 agent có thể làm cặp implementer/reviewer.
