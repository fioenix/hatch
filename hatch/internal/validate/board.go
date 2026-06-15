// Package validate holds board-level checks that span tickets, registry and
// workflow (config-only checks live in the config package).
package validate

import (
	"fmt"
	"regexp"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

// safeID is the allowed shape of a ticket id (also keeps it safe as a path
// segment for branches, worktrees and run transcripts).
var safeID = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Board validates every ticket against the workflow lanes and registry roster.
func Board(ws *config.Workspace, b *store.Board) []config.Problem {
	var probs []config.Problem
	add := func(src, m string) { probs = append(probs, config.Problem{Source: src, Msg: m}) }

	tickets, err := b.List(ws.Workflow.LaneIDs())
	if err != nil {
		add("board", err.Error())
		return probs
	}

	seen := map[string]string{} // id → lane
	for _, t := range tickets {
		src := fmt.Sprintf("board/%s/%s", t.Lane, t.Filename())
		if t.ID == "" {
			add(src, "ticket missing id")
			continue
		}
		if !safeID.MatchString(t.ID) {
			add(src, fmt.Sprintf("unsafe ticket id %q (allowed: letters, digits, . _ -)", t.ID))
		}
		if prev, ok := seen[t.ID]; ok {
			add(src, fmt.Sprintf("duplicate ticket id %q (also in %s)", t.ID, prev))
		}
		seen[t.ID] = t.Lane

		// status must agree with the lane the file lives in.
		if t.Status != "" && t.Status != t.Lane {
			add(src, fmt.Sprintf("status %q disagrees with lane %q", t.Status, t.Lane))
		}
		// role must exist in the registry.
		if t.Role != "" {
			if _, ok := ws.Registry.RoleByID(t.Role); !ok {
				add(src, fmt.Sprintf("unknown role %q", t.Role))
			}
		}
		// a claiming agent must be allowed to hold the ticket's role.
		if t.Claim != nil && t.Claim.Agent != "" {
			a, ok := ws.Registry.AgentByID(t.Claim.Agent)
			if !ok {
				add(src, fmt.Sprintf("claim by unknown agent %q", t.Claim.Agent))
			} else if t.Role != "" && !hasRole(a.Roles, t.Role) {
				add(src, fmt.Sprintf("agent %q may not hold role %q", a.ID, t.Role))
			}
		}
		// dependencies must reference real tickets.
		for _, dep := range t.DependsOn {
			if _, ok := seen[dep]; !ok {
				// dep may appear later; do a second-pass check below.
				_ = ok
			}
		}
	}

	// second pass: dependency existence (now that all ids are known).
	for _, t := range tickets {
		for _, dep := range t.DependsOn {
			if _, ok := seen[dep]; !ok {
				add(fmt.Sprintf("board/%s/%s", t.Lane, t.Filename()),
					fmt.Sprintf("depends_on unknown ticket %q", dep))
			}
		}
	}
	return probs
}

func hasRole(roles []string, r string) bool {
	for _, x := range roles {
		if x == r {
			return true
		}
	}
	return false
}
