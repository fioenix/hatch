package compile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fioenix/hatch/internal/config"
	"github.com/fioenix/hatch/internal/model"
	"github.com/fioenix/hatch/internal/paths"
)

// RoleContent is one role's L1 context: structured binding + the prose body.
type RoleContent struct {
	Role model.Role
	Body string // roles/<id>.md body (empty if file missing)
}

// Bundle is everything needed to render a surface: L0 charter, the L1 roles
// served on that surface, the L2 pointers (not the content), plus the protocol
// the agent self-follows (workflow prose + chat etiquette + DoD).
type Bundle struct {
	Surface          Surface
	Agents           []model.Agent // agents reading this surface
	Charter          string        // L0
	WorkingAgreement string        // L0 — how the squad works professionally
	Roles            []RoleContent // L1
	ContextRefs      []string      // L2 pointers (union of role refs)
	Project          string

	Workflow *model.Workflow // process, rendered as prose (not an engine)
	Policy   model.Policy    // governance toggles surfaced into the DoD
	Lead     *model.Agent    // set when this surface carries the orchestrator agent
}

// stripFrontmatter removes a leading `---`-fenced YAML block, returning the
// Markdown body trimmed of surrounding blank lines.
func stripFrontmatter(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if strings.HasPrefix(s, "---\n") {
		if end := strings.Index(s[4:], "\n---"); end >= 0 {
			rest := s[4+end+len("\n---"):]
			s = strings.TrimPrefix(rest, "\n")
		}
	}
	return strings.TrimSpace(s)
}

// readRoleBody loads roles/<id>.md, returning "" if the file is absent.
func readRoleBody(l paths.Layout, role model.Role) string {
	name := role.File
	if name == "" {
		name = role.ID + ".md"
	}
	raw, err := os.ReadFile(filepath.Join(l.Roles(), name))
	if err != nil {
		return ""
	}
	return stripFrontmatter(string(raw))
}

// leadAgent returns the squad's orchestrator: the explicit registry.orchestrator
// agent (set by `hatch init --client`), else the first agent holding the
// "conductor" role, else the first agent in the roster. This is the agent a user
// typically opens first; its surface gets the orchestrator block.
func leadAgent(ws *config.Workspace) *model.Agent {
	if id := ws.Registry.Orchestrator; id != "" {
		for i := range ws.Registry.Agents {
			if ws.Registry.Agents[i].ID == id {
				return &ws.Registry.Agents[i]
			}
		}
	}
	for i := range ws.Registry.Agents {
		for _, r := range ws.Registry.Agents[i].Roles {
			if r == "conductor" {
				return &ws.Registry.Agents[i]
			}
		}
	}
	if len(ws.Registry.Agents) > 0 {
		return &ws.Registry.Agents[0]
	}
	return nil
}

// buildBundle assembles the bundle for a surface given the role ids it serves.
func buildBundle(ws *config.Workspace, surf Surface, agents []model.Agent, roleIDs []string) Bundle {
	charter := ""
	if raw, err := os.ReadFile(ws.Layout.Charter()); err == nil {
		charter = stripFrontmatter(string(raw))
	}
	workingAgreement := ""
	if raw, err := os.ReadFile(ws.Layout.WorkingAgreement()); err == nil {
		workingAgreement = stripFrontmatter(string(raw))
	}
	sort.Strings(roleIDs)
	var roles []RoleContent
	refSet := map[string]bool{}
	var refs []string
	for _, id := range roleIDs {
		role, ok := ws.Registry.RoleByID(id)
		if !ok {
			role = model.Role{ID: id}
		}
		roles = append(roles, RoleContent{Role: role, Body: readRoleBody(ws.Layout, role)})
		for _, r := range role.ContextRefs {
			if !refSet[r] {
				refSet[r] = true
				refs = append(refs, r)
			}
		}
	}
	// The orchestrator block goes only on the surface that carries the lead.
	var lead *model.Agent
	if la := leadAgent(ws); la != nil {
		for _, a := range agents {
			if a.ID == la.ID {
				lead = la
				break
			}
		}
	}

	return Bundle{
		Surface:          surf,
		Agents:           agents,
		Charter:          charter,
		WorkingAgreement: workingAgreement,
		Roles:            roles,
		ContextRefs:      refs,
		Project:          ws.Registry.Project,
		Workflow:         ws.Workflow,
		Policy:           ws.Registry.Policy,
		Lead:             lead,
	}
}
