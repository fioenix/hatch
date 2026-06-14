package compile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// RoleContent is one role's L1 context: structured binding + the prose body.
type RoleContent struct {
	Role model.Role
	Body string // roles/<id>.md body (empty if file missing)
}

// Bundle is everything needed to render a surface: L0 charter, the L1 roles
// served on that surface, and the L2 pointers (not the content).
type Bundle struct {
	Surface     Surface
	Agents      []model.Agent // agents reading this surface
	Charter     string        // L0
	Roles       []RoleContent // L1
	ContextRefs []string      // L2 pointers (union of role refs)
	Project     string
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

// buildBundle assembles the bundle for a surface given the role ids it serves.
func buildBundle(ws *config.Workspace, surf Surface, agents []model.Agent, roleIDs []string) Bundle {
	charter := ""
	if raw, err := os.ReadFile(ws.Layout.Charter()); err == nil {
		charter = stripFrontmatter(string(raw))
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
	return Bundle{
		Surface:     surf,
		Agents:      agents,
		Charter:     charter,
		Roles:       roles,
		ContextRefs: refs,
		Project:     ws.Registry.Project,
	}
}
