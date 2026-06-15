// Package compile turns the SSOT (charter + roles + context) into the native
// instruction files each coding-agent CLI expects, applying L0+L1+pointer
// layering. See docs/10-agent-adapters.md for the surface table.
package compile

import (
	"path/filepath"
)

// Surface is a distinct instruction-file target. Several agent kinds can share
// one surface (Codex and Antigravity both read AGENTS.md).
type Surface struct {
	Key  string // claude | agents | gemini | kiro
	Desc string
}

// Known surfaces.
var (
	SurfaceClaude = Surface{"claude", "Claude Code — CLAUDE.md"}
	SurfaceAgents = Surface{"agents", "AGENTS.md (Codex / Antigravity / Gemini-compat)"}
	SurfaceGemini = Surface{"gemini", "Gemini CLI — GEMINI.md"}
	SurfaceKiro   = Surface{"kiro", "Kiro — .kiro/steering/"}
)

// surfaceForKind maps a registry agent kind to its default surface key.
// "manual" agents have no compile surface.
func surfaceForKind(kind string) (Surface, bool) {
	switch kind {
	case "claude":
		return SurfaceClaude, true
	case "codex", "antigravity":
		return SurfaceAgents, true
	case "gemini", "agy":
		return SurfaceGemini, true
	case "kiro":
		return SurfaceKiro, true
	default:
		return Surface{}, false
	}
}

// surfaceByKey resolves an explicit surface key from registry agent.Surfaces.
func surfaceByKey(key string) (Surface, bool) {
	switch key {
	case "claude":
		return SurfaceClaude, true
	case "agents", "codex", "antigravity":
		return SurfaceAgents, true
	case "gemini":
		return SurfaceGemini, true
	case "kiro":
		return SurfaceKiro, true
	default:
		return Surface{}, false
	}
}

// OutputPaths returns the files a surface writes, relative to the repo root.
func (s Surface) OutputPaths(repoRoot string) []string {
	switch s.Key {
	case "claude":
		return []string{filepath.Join(repoRoot, "CLAUDE.md")}
	case "agents":
		return []string{filepath.Join(repoRoot, "AGENTS.md")}
	case "gemini":
		return []string{filepath.Join(repoRoot, "GEMINI.md")}
	case "kiro":
		return []string{filepath.Join(repoRoot, ".kiro", "steering", "hatch.md")}
	}
	return nil
}
