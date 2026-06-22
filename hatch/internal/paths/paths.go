// Package paths defines the on-disk layout of a .hatch/ workspace and helpers
// to locate it. The filesystem IS the database, so these paths are the schema.
package paths

import (
	"errors"
	"os"
	"path/filepath"
)

// Dir is the workspace directory name placed at a repository root.
const Dir = ".hatch"

// Layout names within a .hatch/ directory. SSOT (committed) lives at the top
// level; everything regenerable or machine-local lives under RunDir so a single
// gitignore line (/.hatch/run/) covers all runtime state.
const (
	// SSOT — hand-edited, committed.
	CharterFile          = "charter.md"
	WorkingAgreementFile = "working-agreement.md"
	RegistryFile         = "registry.yaml"
	WorkflowFile         = "workflow.yaml"

	RolesDir    = "roles"
	ContextDir  = "context"
	KBDir       = "kb"
	ProtocolDir = "protocol"

	KBIndexFile = "kb/index.md"
	KBMetaFile  = "kb/.meta.json"

	// RunDir holds all runtime state (chat bus, presence, sessions, slack,
	// compiled manifest, logs, ledger, mcp snippets). Gitignored as one dir.
	RunDir = "run"

	BusDir         = "bus"
	BoardDir       = "board"
	RosterFile     = "roster.json"
	SessionsFile   = "sessions.json"
	LedgerDir      = "ledger"
	CompiledDir    = "compiled"
	LogsDir        = "logs"
	SlackDir       = "slack"
	MCPDir         = "mcp"
	ManifestFile   = "compiled/.manifest.json"
	MCPLogFile     = "logs/mcp.jsonl"
	SlackConfigDir = "slack"
)

// Layout resolves absolute paths inside a single workspace root.
type Layout struct{ Root string } // Root is the .hatch directory itself.

func (l Layout) path(parts ...string) string {
	return filepath.Join(append([]string{l.Root}, parts...)...)
}

// run resolves a path under the runtime dir (.hatch/run/...).
func (l Layout) run(parts ...string) string {
	return l.path(append([]string{RunDir}, parts...)...)
}

// SSOT (committed).
func (l Layout) Charter() string          { return l.path(CharterFile) }
func (l Layout) WorkingAgreement() string { return l.path(WorkingAgreementFile) }
func (l Layout) Registry() string         { return l.path(RegistryFile) }
func (l Layout) Workflow() string         { return l.path(WorkflowFile) }
func (l Layout) Roles() string            { return l.path(RolesDir) }
func (l Layout) Context() string          { return l.path(ContextDir) }
func (l Layout) KB() string               { return l.path(KBDir) }
func (l Layout) Protocol() string         { return l.path(ProtocolDir) }
func (l Layout) KBIndex() string          { return l.path(KBIndexFile) }
func (l Layout) KBMeta() string           { return l.path(KBMetaFile) }

// Runtime (under .hatch/run/, gitignored).
func (l Layout) Run() string             { return l.path(RunDir) }
func (l Layout) Bus() string             { return l.run(BusDir) }
func (l Layout) Board() string           { return l.run(BoardDir) }
func (l Layout) Lane(name string) string { return l.run(BoardDir, name) }
func (l Layout) Ledger() string          { return l.run(LedgerDir) }
func (l Layout) Compiled() string        { return l.run(CompiledDir) }
func (l Layout) Logs() string            { return l.run(LogsDir) }
func (l Layout) MCPLog() string          { return l.run(MCPLogFile) }
func (l Layout) Manifest() string        { return l.run(ManifestFile) }
func (l Layout) MCPSnippets() string     { return l.run(MCPDir) }
func (l Layout) Roster() string          { return l.run(RosterFile) }
func (l Layout) Sessions() string        { return l.run(SessionsFile) }
func (l Layout) Slack() string           { return l.run(SlackDir) }
func (l Layout) SlackConfig() string     { return l.run(SlackDir, "config.json") }
func (l Layout) SlackThreadmap() string  { return l.run(SlackDir, "threadmap.json") }
func (l Layout) DocTemplates() string    { return l.path("templates", "docs") }

// SafeSegment sanitizes s for use as a single path segment, preventing path
// traversal: only [A-Za-z0-9._-] survive, the rest become '-', and the
// traversal tokens "", ".", ".." collapse to "_".
func SafeSegment(s string) string {
	b := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b = append(b, r)
		default:
			b = append(b, '-')
		}
	}
	out := string(b)
	if out == "" || out == "." || out == ".." {
		return "_"
	}
	return out
}

func (l Layout) Runs(ticket string) string {
	if ticket == "" || ticket == "-" {
		ticket = "system"
	}
	return l.path("runs", SafeSegment(ticket))
}

// RepoRoot is the directory that contains the .hatch directory.
func (l Layout) RepoRoot() string { return filepath.Dir(l.Root) }

// ErrNotFound indicates no .hatch workspace was located.
var ErrNotFound = errors.New("no .hatch workspace found (run `hatch init`)")

// GlobalRoot is the user-level .hatch directory used as the default when no
// local .hatch overrides it: $HATCH_HOME if set (taken as the .hatch path
// directly), else ~/.hatch. Returns "" if the home dir can't be resolved.
func GlobalRoot() string {
	if v := os.Getenv("HATCH_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, Dir)
}

// FindLocal walks up from start looking for a .hatch directory and returns its
// Layout. It stops at the filesystem root. It does NOT fall back to the global
// workspace.
func FindLocal(start string) (Layout, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return Layout{}, err
	}
	for {
		cand := filepath.Join(dir, Dir)
		if fi, err := os.Stat(cand); err == nil && fi.IsDir() {
			return Layout{Root: cand}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Layout{}, ErrNotFound
		}
		dir = parent
	}
}

// Find resolves the workspace the way commands should: a local .hatch (nearest
// ancestor of start) overrides the global ~/.hatch. Returns the local one if
// present, else the global one if it exists, else ErrNotFound.
func Find(start string) (Layout, error) {
	if l, err := FindLocal(start); err == nil {
		return l, nil
	}
	if g := GlobalRoot(); g != "" {
		if fi, err := os.Stat(g); err == nil && fi.IsDir() {
			return Layout{Root: g}, nil
		}
	}
	return Layout{}, ErrNotFound
}

// At returns a Layout rooted at <dir>/.hatch without requiring it to exist.
func At(dir string) Layout {
	return Layout{Root: filepath.Join(dir, Dir)}
}
