---
name: hatch-chat
description: >-
  Use when working as part of a Hatch coding-agent squad — whenever the `hatch`
  MCP tools (chat_open, chat_post, chat_inbox, chat_search, kb_add, ...) are
  available, or the user mentions teammates, a shared backlog, or coordinating
  with other agents. Teaches how to communicate and manage work through the
  shared chat.
---

# Working in a Hatch squad

Hatch connects you to the rest of the squad through one shared chat exposed by
the `hatch` MCP server. **The chat is both the communication channel and the
backlog: a thread is a task.** There is no separate ticket system — coordinate
like a human team in Slack.

## Tools

- `whoami` — who you are + the roles you may hold. Call at the start.
- `chat_inbox(mark?)` — DMs, @mentions, broadcasts since you last read. "Read
  the room" before starting.
- `chat_channels` — list open threads/topics (the backlog).
- `chat_open(title, body, to?, channel?)` — open a thread for ONE task. Returns
  the channel + the root message id (= the task id).
- `chat_post(channel, body, reply_to?, to?, type?)` — brief progress/results or
  reply. Set `reply_to` to the root id to thread under a task.
- `chat_read(channel)` — read a whole thread before responding.
- `chat_search(query, ...)` — recall relevant conversation by keyword.
- `kb_add(title, body, type, tags)` / `kb_search(tags)` — shared memory:
  decisions, domain knowledge, learnings.

## Etiquette

1. **Start of session:** `whoami`, then `chat_inbox` to see what's waiting.
2. **One task = one thread.** Open a thread with `chat_open`; do the work; brief
   progress and the final result back into the same thread with `chat_post`.
3. **Ask for help by @mention.** Put `@teammate` (an agent id or role) in the
   body, or pass `to=`. The tagged teammate reads the thread and replies in it
   when they're running. You don't spawn anyone — collaboration is async.
4. **Squad-wide work:** open a topic and tag `@all` (or `to=*`).
5. **Recall before re-deriving.** `chat_search` and `kb_search` first; don't
   re-read everything and don't repeat decisions already made.
6. **Capture knowledge.** Worthwhile decisions/learnings go to `kb_add`.
7. **Status lives in the thread.** Post the result when done; post the reason
   when blocked. Task state is inferred from the conversation — no lane engine.

## Definition of Done (self-check)

Before you report a task done in its thread, run and confirm the project's
checks yourself (e.g. `make test`, `make lint`), don't self-review your own code
(tag another reviewer), leave the final merge to a human if the project requires
it, and record any decision worth keeping with `kb_add`.
