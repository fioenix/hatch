package mcpserver

// Tool input/output schemas. The SDK generates JSON Schema from these structs;
// json tags are the wire names agents see.

type openIn struct {
	Channel string `json:"channel,omitempty" jsonschema:"channel/topic id (e.g. #design, T-12); empty = derive from title"`
	Title   string `json:"title,omitempty" jsonschema:"short task/topic title"`
	Body    string `json:"body" jsonschema:"first message; use @name to tag teammates"`
	To      string `json:"to,omitempty" jsonschema:"comma-separated recipients: agent/role/#channel/*"`
}

type postIn struct {
	Channel string `json:"channel" jsonschema:"channel/topic id to post into"`
	Body    string `json:"body" jsonschema:"message body; use @name to tag teammates"`
	To      string `json:"to,omitempty" jsonschema:"comma-separated recipients/mentions"`
	ReplyTo string `json:"reply_to,omitempty" jsonschema:"root message id to thread under"`
	Type    string `json:"type,omitempty" jsonschema:"msg|ask|reply|decision (default msg)"`
}

type postOut struct {
	Channel   string `json:"channel"`
	MessageID string `json:"message_id"`
}

type channelIn struct {
	Channel string `json:"channel" jsonschema:"channel/topic id to read"`
}

type inboxIn struct {
	Mark bool `json:"mark,omitempty" jsonschema:"true = advance read cursor after reading"`
}

type searchIn struct {
	Query   string `json:"query,omitempty"`
	Channel string `json:"channel,omitempty"`
	From    string `json:"from,omitempty"`
	Type    string `json:"type,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type kbAddIn struct {
	Type  string `json:"type,omitempty" jsonschema:"decision|domain|learning (default learning)"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Tags  string `json:"tags,omitempty" jsonschema:"comma-separated tags"`
}

type kbSearchIn struct {
	Tags string `json:"tags,omitempty" jsonschema:"comma-separated tags (empty = all)"`
}

type whoamiOut struct {
	Agent string   `json:"agent"`
	Roles []string `json:"roles"`
}

type joinIn struct {
	Kind      string `json:"kind,omitempty" jsonschema:"agent kind: claude|codex|agy|kiro|mock|user (default: from registry)"`
	Roles     string `json:"roles,omitempty" jsonschema:"comma-separated role ids held in the room (default: from registry)"`
	SessionID string `json:"session_id,omitempty" jsonschema:"resumable session id holding your memory, so teammates can wake the same you"`
	Note      string `json:"note,omitempty" jsonschema:"optional status note"`
}

type joinOut struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type rosterOut struct {
	Members []string `json:"members"`
}

type textOut struct {
	Text string `json:"text"`
}

type messagesOut struct {
	Messages []string `json:"messages"`
}

type channelsOut struct {
	Channels []string `json:"channels"`
}

type kbAddOut struct {
	ID string `json:"id"`
}

type kbSearchOut struct {
	Entries []string `json:"entries"`
}
