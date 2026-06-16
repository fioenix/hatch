package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/model"
)

// Result reports what a compile run produced.
type Result struct {
	Written []string // output paths written (relative to repo root not guaranteed)
	Bundles []Bundle
}

// surfacePlan groups, per surface key, the agents and the union of their roles.
type surfacePlan struct {
	surface Surface
	agents  []model.Agent
	roleIDs []string
}

// plan computes which surfaces to emit from the registry roster.
func plan(ws *config.Workspace) ([]surfacePlan, []string) {
	bySurface := map[string]*surfacePlan{}
	var warnings []string

	add := func(surf Surface, a model.Agent) {
		sp := bySurface[surf.Key]
		if sp == nil {
			sp = &surfacePlan{surface: surf}
			bySurface[surf.Key] = sp
		}
		sp.agents = append(sp.agents, a)
		seen := map[string]bool{}
		for _, r := range sp.roleIDs {
			seen[r] = true
		}
		for _, r := range a.Roles {
			if !seen[r] {
				sp.roleIDs = append(sp.roleIDs, r)
				seen[r] = true
			}
		}
	}

	for _, a := range ws.Registry.Agents {
		keys := a.Surfaces
		if len(keys) == 0 {
			if surf, ok := surfaceForKind(a.Kind); ok {
				add(surf, a)
			} else if a.Kind != "manual" && a.Kind != "shell" {
				warnings = append(warnings, fmt.Sprintf("agent %q kind %q has no compile surface", a.ID, a.Kind))
			}
			continue
		}
		for _, k := range keys {
			if surf, ok := surfaceByKey(k); ok {
				add(surf, a)
			} else {
				warnings = append(warnings, fmt.Sprintf("agent %q has unknown surface %q", a.ID, k))
			}
		}
	}

	var plans []surfacePlan
	for _, sp := range bySurface {
		plans = append(plans, *sp)
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].surface.Key < plans[j].surface.Key })
	return plans, warnings
}

// Run compiles the SSOT to every surface and updates the manifest. Outputs go
// to ws.Out() (the working repo), which may differ from the SSOT location when
// the global ~/.hatch is in use.
func Run(ws *config.Workspace) (*Result, []string, error) {
	repoRoot := ws.Out()
	plans, warnings := plan(ws)

	m := &Manifest{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Sources:     sourceHashes(ws.Layout),
		Outputs:     map[string]string{},
	}
	res := &Result{}

	for _, sp := range plans {
		b := buildBundle(ws, sp.surface, sp.agents, sp.roleIDs)
		content := []byte(Render(b))
		for _, out := range sp.surface.OutputPaths(repoRoot) {
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return nil, warnings, err
			}
			if err := os.WriteFile(out, content, 0o644); err != nil {
				return nil, warnings, err
			}
			rel, _ := filepath.Rel(repoRoot, out)
			m.Outputs[filepath.ToSlash(rel)] = hashBytes(content)
			res.Written = append(res.Written, out)
		}
		res.Bundles = append(res.Bundles, b)
	}

	// Register the Hatch MCP server with each agent so it can reach the shared
	// chat + KB under its own identity (the embedded-harness integration).
	mcpFiles, err := writeMCPConfigs(ws, repoRoot)
	if err != nil {
		return nil, warnings, err
	}
	res.Written = append(res.Written, mcpFiles...)

	if err := m.Save(ws.Layout); err != nil {
		return nil, warnings, err
	}
	return res, warnings, nil
}
