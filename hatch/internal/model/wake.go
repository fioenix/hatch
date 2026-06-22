package model

// WakeReason explains why the delivery layer woke a member. Every wake is the
// consequence of a message someone actively sent — never a scheduler inventing
// work. This is the line between "chat server" (delivery) and "boss"
// (work-orchestration): Hatch does the former, never the latter.
type WakeReason string

const (
	WakeMention  WakeReason = "mention"           // a message addressed/@-mentioned the member
	WakeReplyAsk WakeReason = "reply_to_open_ask" // a reply landed on the member's open question
	WakeDM       WakeReason = "dm"                // a direct message to the member
	WakeNudge    WakeReason = "nudge"             // a one-shot reminder to a stalled owner
)

// WakeDecision is the delivery layer's instruction to wake one member with the
// message(s) that triggered it. Multiple triggering messages are coalesced into
// Payload. A non-empty Hold means the wake is deferred, not cancelled.
type WakeDecision struct {
	Agent   string     `json:"agent"`
	Reason  WakeReason `json:"reason"`
	Payload []Message  `json:"payload"`
	Hold    HoldReason `json:"hold,omitempty"` // "" = deliver now
}

// HoldReason defers a wake without dropping it. The daemon redelivers held
// wakes when the condition clears (the member finishes its turn, the rate
// window opens). Holding — never dropping — keeps the liveness contract: a
// message addressed to a teammate is always eventually delivered.
type HoldReason string

const (
	HoldNone    HoldReason = ""
	HoldWorking HoldReason = "working"    // member is mid-turn; deliver at its next turn boundary
	HoldRate    HoldReason = "rate_limit" // per-member wake cap hit; deliver when the window opens
)

// Escalation routes a coordination problem to the boss (the human) when the
// squad cannot safely keep going on its own. It is the circuit-breaker behind
// the Working Agreement: depth/loop limits and stalled ownership surface here
// instead of burning tokens in silence.
type Escalation struct {
	Episode string `json:"episode"` // thread-root id the problem belongs to
	Cause   string `json:"cause"`   // see escalation cause constants
	To      string `json:"to"`      // usually the boss/user id
	Note    string `json:"note"`
}

// Escalation causes.
const (
	EscalateDepthLimit   = "depth_limit"   // auto-wake cascade exceeded the configured depth
	EscalateLoopBreak    = "loop_break"    // a pair ping-ponged without progress
	EscalateStalledOwner = "stalled_owner" // a thread owner went silent past the window
)
