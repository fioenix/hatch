package model

// Workflow is a *prose* description of the squad's process: it compiles into the
// instruction surfaces so agents self-follow it via chat. It is not an engine —
// there are no tickets or lane directories; a task is a bus thread and its state
// is inferred from the conversation.

// Gate kinds, rendered as prose guidance (not enforced by an engine).
const (
	GateCommand   = "command"        // run a shell command, exit 0 = pass
	GateChecklist = "checklist"      // a DoD checklist referenced by file
	GateRequired  = "required-field" // a ticket frontmatter field must be set
	GatePolicy    = "policy"         // a registry policy toggle
	GateHuman     = "human-gate"     // stop and wait for a human
)

// Gate is a condition checked when a transition declares it.
type Gate struct {
	Type  string `yaml:"type"`
	Run   string `yaml:"run,omitempty"`   // command gates
	Ref   string `yaml:"ref,omitempty"`   // checklist/policy gates
	Field string `yaml:"field,omitempty"` // required-field gates
}

// Lane is a stage a task moves through (backlog → in-progress → review → done),
// described as prose — not a board directory.
type Lane struct {
	ID       string `yaml:"id"`
	WIPLimit int    `yaml:"wip-limit,omitempty"`
	Side     bool   `yaml:"side,omitempty"` // off the main flow (e.g. blocked)
}

// Transition is the only thing that authorises a lane change.
type Transition struct {
	From   string   `yaml:"from"`             // lane id or "*"
	To     string   `yaml:"to"`               // lane id
	By     []string `yaml:"by"`               // role ids allowed, or ["*"]
	Action string   `yaml:"action,omitempty"` // ledger action recorded
	Gates  []string `yaml:"gates,omitempty"`  // gate names that must pass
}

// Ceremony is a recurring coordination event.
type Ceremony struct {
	By      string   `yaml:"by"`
	Trigger string   `yaml:"trigger,omitempty"`
	Actions []string `yaml:"actions,omitempty"`
}

// SpecConfig configures the optional spec-driven (PRD→Design→Tasks) flow.
type SpecConfig struct {
	RequiredFor []string          `yaml:"required-for,omitempty"`
	Artifacts   []string          `yaml:"artifacts,omitempty"`
	Gates       map[string]string `yaml:"gates,omitempty"`
}

// Workflow mirrors spec/workflow.schema.md: a per-project, editable process.
type Workflow struct {
	Version     int                 `yaml:"version"`
	Template    string              `yaml:"template,omitempty"`
	Lanes       []Lane              `yaml:"lanes"`
	Transitions []Transition        `yaml:"transitions"`
	Gates       map[string]Gate     `yaml:"gates,omitempty"`
	Ceremonies  map[string]Ceremony `yaml:"ceremonies,omitempty"`
	Spec        *SpecConfig         `yaml:"spec,omitempty"`
}

// LaneIDs returns lane ids in declaration order.
func (w *Workflow) LaneIDs() []string {
	ids := make([]string, len(w.Lanes))
	for i, l := range w.Lanes {
		ids[i] = l.ID
	}
	return ids
}

// HasLane reports whether a lane id is defined.
func (w *Workflow) HasLane(id string) bool {
	for _, l := range w.Lanes {
		if l.ID == id {
			return true
		}
	}
	return false
}

// LaneByID returns the lane with the given id.
func (w *Workflow) LaneByID(id string) (Lane, bool) {
	for _, l := range w.Lanes {
		if l.ID == id {
			return l, true
		}
	}
	return Lane{}, false
}

// FindTransition returns the transition matching a from→to move, honouring the
// "*" wildcard on From.
func (w *Workflow) FindTransition(from, to string) (Transition, bool) {
	for _, t := range w.Transitions {
		if t.To == to && (t.From == from || t.From == "*") {
			return t, true
		}
	}
	return Transition{}, false
}
