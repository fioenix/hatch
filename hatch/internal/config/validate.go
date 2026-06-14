package config

import (
	"fmt"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

// Problem is a single validation finding.
type Problem struct {
	Source string // file or component
	Msg    string
}

func (p Problem) String() string { return fmt.Sprintf("%s: %s", p.Source, p.Msg) }

// ValidateRegistry checks internal consistency of the roster.
func ValidateRegistry(r *model.Registry) []Problem {
	var probs []Problem
	add := func(f, m string) { probs = append(probs, Problem{f, m}) }

	if r.Version == 0 {
		add("registry.yaml", "missing version")
	}
	seenRole := map[string]bool{}
	for _, role := range r.Roles {
		if role.ID == "" {
			add("registry.yaml", "role with empty id")
			continue
		}
		if seenRole[role.ID] {
			add("registry.yaml", fmt.Sprintf("duplicate role id %q", role.ID))
		}
		seenRole[role.ID] = true
	}
	// reporting lines must reference real roles and not cycle.
	for _, role := range r.Roles {
		if role.ReportsTo == "" {
			continue
		}
		if !seenRole[role.ReportsTo] {
			add("registry.yaml", fmt.Sprintf("role %q reports_to unknown role %q", role.ID, role.ReportsTo))
			continue
		}
		if orgCycle(r, role.ID) {
			add("registry.yaml", fmt.Sprintf("reports_to cycle involving role %q", role.ID))
		}
	}
	seenAgent := map[string]bool{}
	for _, a := range r.Agents {
		if a.ID == "" {
			add("registry.yaml", "agent with empty id")
			continue
		}
		if seenAgent[a.ID] {
			add("registry.yaml", fmt.Sprintf("duplicate agent id %q", a.ID))
		}
		seenAgent[a.ID] = true
		if a.Kind == "" {
			add("registry.yaml", fmt.Sprintf("agent %q missing kind", a.ID))
		}
		for _, rl := range a.Roles {
			if !seenRole[rl] {
				add("registry.yaml", fmt.Sprintf("agent %q references unknown role %q", a.ID, rl))
			}
		}
	}
	return probs
}

// orgCycle reports whether following reports_to from start loops.
func orgCycle(r *model.Registry, start string) bool {
	seen := map[string]bool{}
	cur := start
	for cur != "" {
		if seen[cur] {
			return true
		}
		seen[cur] = true
		role, ok := r.RoleByID(cur)
		if !ok {
			return false
		}
		cur = role.ReportsTo
	}
	return false
}

// ValidateWorkflow checks lanes, transitions and gates refer to each other
// consistently and that the board can reach a terminal lane.
func ValidateWorkflow(w *model.Workflow) []Problem {
	var probs []Problem
	add := func(m string) { probs = append(probs, Problem{"workflow.yaml", m}) }

	if w.Version == 0 {
		add("missing version")
	}
	if len(w.Lanes) == 0 {
		add("no lanes defined")
		return probs
	}
	seenLane := map[string]bool{}
	for _, l := range w.Lanes {
		if l.ID == "" {
			add("lane with empty id")
			continue
		}
		if seenLane[l.ID] {
			add(fmt.Sprintf("duplicate lane id %q", l.ID))
		}
		seenLane[l.ID] = true
	}
	for i, t := range w.Transitions {
		if t.From != "*" && !seenLane[t.From] {
			add(fmt.Sprintf("transition[%d] from unknown lane %q", i, t.From))
		}
		if !seenLane[t.To] {
			add(fmt.Sprintf("transition[%d] to unknown lane %q", i, t.To))
		}
		if len(t.By) == 0 {
			add(fmt.Sprintf("transition[%d] (%s→%s) has no `by` roles", i, t.From, t.To))
		}
		for _, g := range t.Gates {
			if _, ok := w.Gates[g]; !ok {
				add(fmt.Sprintf("transition[%d] references undefined gate %q", i, g))
			}
		}
	}
	for name, g := range w.Gates {
		switch g.Type {
		case model.GateCommand:
			if g.Run == "" {
				add(fmt.Sprintf("gate %q (command) missing `run`", name))
			}
		case model.GateRequired:
			if g.Field == "" {
				add(fmt.Sprintf("gate %q (required-field) missing `field`", name))
			}
		case model.GateChecklist, model.GatePolicy, model.GateHuman:
			// ref optional / not required
		case "":
			add(fmt.Sprintf("gate %q missing type", name))
		default:
			add(fmt.Sprintf("gate %q has unknown type %q", name, g.Type))
		}
	}
	// Reachability: every non-side lane should reach a lane with no outgoing
	// transition (a terminal/done lane).
	if !reachesTerminal(w) {
		add("no lane graph path reaches a terminal (done) lane")
	}
	return probs
}

func reachesTerminal(w *model.Workflow) bool {
	outgoing := map[string]bool{}
	for _, t := range w.Transitions {
		if t.From != "*" {
			outgoing[t.From] = true
		}
	}
	for _, l := range w.Lanes {
		if l.Side {
			continue
		}
		if !outgoing[l.ID] {
			return true // this lane is terminal
		}
	}
	return false
}

// ValidateCrossRefs checks links that span both files, e.g. transition roles
// must exist in the registry.
func ValidateCrossRefs(r *model.Registry, w *model.Workflow) []Problem {
	var probs []Problem
	roleOK := map[string]bool{"*": true}
	for _, role := range r.Roles {
		roleOK[role.ID] = true
	}
	for i, t := range w.Transitions {
		for _, by := range t.By {
			if !roleOK[by] {
				probs = append(probs, Problem{
					"workflow.yaml",
					fmt.Sprintf("transition[%d] (%s→%s) `by` role %q not in registry", i, t.From, t.To, by),
				})
			}
		}
	}
	return probs
}

// ValidateAll runs every check for a loaded workspace.
func (ws *Workspace) Validate() []Problem {
	var probs []Problem
	probs = append(probs, ValidateRegistry(ws.Registry)...)
	probs = append(probs, ValidateWorkflow(ws.Workflow)...)
	probs = append(probs, ValidateCrossRefs(ws.Registry, ws.Workflow)...)
	return probs
}
