# Hatch chat-native model (the SoT contract)

Hatch has its **own** chat substrate — the **bus** (`.hatch/run/bus/`). It is the
single source of truth for all squad communication. Slack, Discord, or any other
client is an **interchangeable bridge**: a thin projection of this model, never
the source. Hatch does not ship a chat UI; it only guarantees this model is
structurally sound, following Slack's conversation primitives so any bridge maps
cleanly.

## Primitives

- **Workspace** — one `.hatch` room = one repo. All channels live in it.
- **Channel** — a named conversation, addressed by id (`design`, `task-42`,
  `dm-claude-codex`). Stored as one append-only log. Listed by `Channels()`.
- **Message** — one turn:
  `{ID, Channel, TS, From, To[], Type, InReplyTo, Body}`. `ID` is unique;
  `(Channel, ID)` addresses a message. `From` = sender member id.
- **Thread** — a reply chain. A message with `InReplyTo = R` belongs to the
  thread rooted at message `R`. One level deep, exactly like Slack's `thread_ts`
  (thread id = root message id).
- **Mention** — `@handle` in the body (agent id, role id, or `*`/`all`), merged
  into `To[]`. This is the routing + wake key: the daemon wakes whoever a
  message addresses.
- **Type** (hatch extension) — `msg | ask | reply | decision`. Slack has no
  equivalent; bridges may ignore it or map it to formatting. `ask` drives the
  open-question / wake semantics.

## Deliberately omitted (YAGNI)

Reactions, edits/deletes, rich blocks, channel metadata (topic/members),
presence-in-channel. Add one only when a concrete bridge needs it — not before.

## Bridge mapping (how a client projects the model)

A bridge implements: list channels, read a channel's messages, post a message
(from + thread + mentions), and ingest inbound human messages back onto the bus.
Everything it needs is in the model above.

| Bus | Slack (current policy) | Discord |
|---|---|---|
| channel | thread under one `#squad` *(or* a channel, faithful*)* | thread or channel |
| message | message | message |
| `InReplyTo` | `thread_ts` | message reference / thread |
| `From` | bot username/icon (per-agent token) | webhook username |
| `@handle` | literal text or `<@U…>` | `<@id>` or text |
| `Type` | (ignored / formatting) | (ignored / formatting) |

The Slack bridge collapses each bus channel into a thread under one `#squad`
(chosen topology); a faithful mapping would be channel↔channel. Both are valid
**bridge policies** — the native model does not change.

## Invariants

- Agents talk to the **bus** (via MCP `chat_post`/`chat_read`), never to a client
  directly. The bridge projects to/from the client.
- The bus works with **no bridge** — two agents coordinate through it offline.
- A bridge is replaceable (Slack today, Discord tomorrow) without touching the
  model, the daemon, or the agents.
