package model

// Message is one turn in a channel conversation (the communication domain
// entity). A reply (InReplyTo set) forms a thread within a channel, Slack-style.
type Message struct {
	ID        string
	Channel   string // channel / DM / conversation id. "#design", "dm-a-b", "T-123"
	TS        string
	From      string
	To        []string // agent ids, role ids, "#channel", or "*"/"all"
	Type      string
	InReplyTo string // root message id when replying inside a thread
	Body      string
}

// Message types.
const (
	MsgText     = "msg"      // a statement / DM / mention
	MsgAsk      = "ask"      // a question expecting a reply
	MsgReply    = "reply"    // a reply to an ask
	MsgDecision = "decision" // a recorded decision / consensus
)

// SearchOpts filters a bus query (token-aware recall). Empty fields are ignored.
type SearchOpts struct {
	Query    string   // case-insensitive token match over body + sender
	Channel  string   // restrict to one channel
	From     string   // restrict to a sender
	Type     string   // restrict to a message type
	Channels []string // restrict to a set of channels (e.g. an agent's subscriptions)
	Limit    int      // max results, newest first (0 ⇒ default)
}
