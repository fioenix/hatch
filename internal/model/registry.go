package model

// Agent is an execution entity (a coding-agent CLI) the team can assign work
// to. Capabilities and the adapter that drives it are declared here.
type Agent struct {
	ID       string   `yaml:"id"`
	Kind     string   `yaml:"kind"`               // adapter key: claude | codex | agy | kiro | mock | manual
	Roles    []string `yaml:"roles"`              // role ids this agent may hold
	Cmd      string   `yaml:"cmd,omitempty"`      // executable name (defaults per kind)
	Model    string   `yaml:"model,omitempty"`    // model override passed to the adapter
	Surfaces []string `yaml:"surfaces,omitempty"` // compile targets this agent reads (claude, codex, ...)
	WIP      int      `yaml:"wip,omitempty"`      // max concurrent tickets (0 = unlimited)
	Sandbox  string   `yaml:"sandbox,omitempty"`  // capability hint for orchestrator (e.g. workspace-write)
	Approval string   `yaml:"approval,omitempty"` // approval/permission mode hint
	// AuthCheck is a non-mutating command (argv) that exits 0 when the agent CLI
	// is authenticated, e.g. "login status". `hatch doctor` runs it instead of
	// inspecting credential files. Overrides the per-kind default.
	AuthCheck []string `yaml:"auth_check,omitempty"`

	BudgetUSD   float64 `yaml:"budget_usd,omitempty"`    // "salary": cost ceiling per cycle (tracked, not enforced)
	RatePerMTok float64 `yaml:"rate_per_mtok,omitempty"` // USD per 1M tokens, for cost estimate when provider gives only tokens
}

// Authority captures a role's delegation-of-authority limits (docs/14).
type Authority struct {
	CanApprove         bool     `yaml:"can_approve,omitempty"`
	BudgetAuthorityUSD float64  `yaml:"budget_authority_usd,omitempty"`
	DecisionScope      []string `yaml:"decision_scope,omitempty"`
}

// Role is a bundle of responsibilities + boundaries + the L1 context to load.
// The prose lives in roles/<id>.md; this is the structured binding metadata.
type Role struct {
	ID          string     `yaml:"id"`
	Title       string     `yaml:"title,omitempty"`
	File        string     `yaml:"file,omitempty"`         // roles/<id>.md (defaulted)
	ContextRefs []string   `yaml:"context_refs,omitempty"` // L1 SSOT paths compiled in
	ReportsTo   string     `yaml:"reports_to,omitempty"`   // parent role in the org chart
	Authority   *Authority `yaml:"authority,omitempty"`
}

// Policy captures team-wide governance toggles enforced at gates.
type Policy struct {
	NoSelfReview  bool     `yaml:"no_self_review"`
	HumanMerge    bool     `yaml:"human_merge"`
	ProtectGlobs  []string `yaml:"protect,omitempty"`         // paths agents may not touch
	EscalateTo    string   `yaml:"escalate_to,omitempty"`     // role/agent to escalate to (default conductor)
	TeamBudgetUSD float64  `yaml:"team_budget_usd,omitempty"` // team cost ceiling per cycle (tracked, not enforced)
}

// WorkflowRef points the registry at the per-project workflow definition.
type WorkflowRef struct {
	Ref string `yaml:"ref,omitempty"`
}

// KBConfig configures the Knowledge Base backend (see docs/15).
type KBConfig struct {
	Mode      string `yaml:"mode,omitempty"`      // native | obsidian
	Vault     string `yaml:"vault,omitempty"`     // path (rel to repo) or "" = .hatch/kb
	Wikilinks bool   `yaml:"wikilinks,omitempty"` // render links as [[wikilinks]]
}

// Registry mirrors spec/registry.schema.md: the team roster and bindings.
type Registry struct {
	Version int    `yaml:"version"`
	Project string `yaml:"project,omitempty"`
	// Orchestrator is the agent id that holds the conductor seat for this
	// workspace; set by `hatch init --client`. Empty = fall back to whichever
	// agent holds the "conductor" role.
	Orchestrator string      `yaml:"orchestrator,omitempty"`
	Roles        []Role      `yaml:"roles"`
	Agents       []Agent     `yaml:"agents"`
	Policy       Policy      `yaml:"policy"`
	Workflow     WorkflowRef `yaml:"workflow,omitempty"`
	KB           KBConfig    `yaml:"kb,omitempty"`
}

// AgentByID returns the agent with the given id, if present.
func (r *Registry) AgentByID(id string) (Agent, bool) {
	for _, a := range r.Agents {
		if a.ID == id {
			return a, true
		}
	}
	return Agent{}, false
}

// RoleByID returns the role with the given id, if present.
func (r *Registry) RoleByID(id string) (Role, bool) {
	for _, role := range r.Roles {
		if role.ID == id {
			return role, true
		}
	}
	return Role{}, false
}

// AgentsForRole lists agents eligible to hold a role.
func (r *Registry) AgentsForRole(role string) []Agent {
	var out []Agent
	for _, a := range r.Agents {
		for _, rl := range a.Roles {
			if rl == role {
				out = append(out, a)
				break
			}
		}
	}
	return out
}
