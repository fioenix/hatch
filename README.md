# overclaud

Optimize how Claude works for you — across every surface.

> **Đang nâng cấp lên đa-agent → [Hatch](hatch/README.md).** overclaud (single-agent) đang được nâng thành một harness điều phối **nhiều coding agent** (Claude Code, Codex, Kiro, Antigravity) trên cùng một repo, tối ưu token, theo quy trình kiểu Agile. overclaud hiện tại trở thành compiler backend cho Claude bên trong Hatch. Sản phẩm thuộc hệ sinh thái **Finolabs**. Xem bộ thiết kế ở [`hatch/`](hatch/README.md).

Claude reads your instructions every message, but most users either don't set them up, put everything in one place, or waste tokens on vague guidance. **overclaud** fixes that.

## What This Is

A complete instruction optimization system for Claude:

- **Templates** — Ready-to-use instructions for 5 roles and 6 project types
- **Layer architecture** — Know exactly where to put each instruction
- **Token optimization** — Every word earns its place
- **Installable skill** — Let Claude set everything up for you

## The Problem

Claude has **9+ surfaces** where instructions can live. Each has different scope, precedence, and purpose:

```
Managed Policy     →  Org-wide (IT-deployed)
User CLAUDE.md     →  Personal, all projects     ← most users stop here
User Rules         →  Personal, path-scoped
Project CLAUDE.md  →  Team-shared, one project
Project Rules      →  Team-shared, path-scoped
Subdirectory       →  On-demand loading
Personal Prefs     →  Chat & Cowork settings
Cowork Instructions→  Per Cowork project
System Prompt      →  API / automation
```

Most users dump everything into one CLAUDE.md. The result: bloated context, wrong scope, wasted tokens every single message.

## Quick Start

### Option 1: Install as a Skill (Recommended)

Build the skill bundle from source (or grab it from [Releases](https://github.com/fioenix/overclaud/releases)):

```bash
make skill   # packages skill/ → overclaud.skill
```

Then install it:

**Claude Code:**
```bash
claude install-skill overclaud.skill
```

**Cowork:** Drop the `.skill` file into your Cowork session.

Then tell Claude: **"Set up my Claude instructions"**

Claude will ask about your role, use cases, and preferences — then generate and install optimized instructions in the right places.

### Option 2: Bootstrap Prompt (Any Surface)

Copy this into any Claude conversation:

```
Fetch and follow the setup guide at:
https://raw.githubusercontent.com/Fioenix/overclaud/main/skill/SKILL.md

Read the reference files in skill/references/ and templates in skill/templates/.
Ask me about my role and preferences, then help me set up optimized Claude instructions.
```

### Option 3: Auto-Suggest After /init (Claude Code)

During setup, the skill offers to install a hook that auto-suggests optimization whenever you
run `/init` on a new project. Claude handles the installation — no manual config needed.

```
You: set up my Claude instructions
Claude: [runs overclaud skill, sets everything up]
Claude: "Want me to enable auto-optimization? Next time you /init a project,
         I'll suggest optimizing the generated CLAUDE.md automatically."
You: yes
Claude: ✓ Hook installed. You're all set.
```

### Option 4: Browse and Copy

Explore the `skill/templates/` directory and copy what fits:

| Need | Browse |
|---|---|
| Global instructions by role | [`templates/global/`](skill/templates/global/) |
| Project-level by use case | [`templates/project/`](skill/templates/project/) |
| Cowork project instructions | [`templates/cowork/`](skill/templates/cowork/) |
| Personal Preferences text | [`templates/personal-preferences/`](skill/templates/personal-preferences/) |
| Path-scoped rules | [`templates/rules/`](skill/templates/rules/) |

## Instruction Layer Map

Understanding where to put instructions is half the battle:

| What to Configure | Where to Put It | Why There |
|---|---|---|
| Your role, communication style, language | `~/.claude/CLAUDE.md` | Applies everywhere, personal |
| Code style for a specific project | `.claude/rules/code-style.md` | Scoped to project, path-specific |
| Team coding standards | `./.claude/CLAUDE.md` | Shared via git |
| Your preference for deep-dive answers | Personal Preferences (Settings) | Applies in Chat & Cowork |
| Cowork daily ops workflow | Cowork Project Instructions | Scoped to one Cowork project |

See [`skill/references/layers.md`](skill/references/layers.md) for the complete architecture.

## Token Optimization

Your global CLAUDE.md loads into every message. At 100 messages/day:

| Extra tokens | Daily waste | Monthly waste |
|---|---|---|
| 50 | 5,000 | 150,000 |
| 200 | 20,000 | 600,000 |
| 500 | 50,000 | 1,500,000 |

**overclaud** templates are designed to maximize signal per token. See [`skill/references/token-optimization.md`](skill/references/token-optimization.md) for techniques.

## Repo Structure

```
overclaud/
├── README.md
├── skill/
│   ├── SKILL.md                    # Installable skill — the setup agent
│   ├── templates/
│   │   ├── global/                 # Global CLAUDE.md by role (5 templates)
│   │   ├── project/                # Project CLAUDE.md by use case (6 templates)
│   │   ├── cowork/                 # Cowork project instructions (3 templates)
│   │   ├── personal-preferences/   # Settings UI text (3 templates)
│   │   └── rules/                  # .claude/rules/ examples (3 templates)
│   └── references/
│       ├── layers.md               # Complete layer architecture
│       ├── token-optimization.md   # Token efficiency techniques
│       └── common-mistakes.md      # Anti-patterns to avoid
├── guides/
│   ├── layering-strategy.md        # What goes where — decision guide
│   └── contributing-templates.md   # How to contribute new templates
└── hatch/                          # Multi-agent upgrade (design docs)
    ├── README.md                   # Hatch — multi-agent harness overview
    ├── docs/                       # 00-vision … 08-roadmap
    └── spec/                       # registry / ticket / ledger schemas
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Template contributions are especially welcome** — the more roles and use cases covered, the more useful this becomes for the community.

## License

MIT — use it however you want.
