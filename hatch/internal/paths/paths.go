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

// Layout names within a .hatch/ directory.
const (
	CharterFile  = "charter.md"
	RegistryFile = "registry.yaml"
	WorkflowFile = "workflow.yaml"

	RolesDir    = "roles"
	ContextDir  = "context"
	KBDir       = "kb"
	BoardDir    = "board"
	LedgerDir   = "ledger"
	ProtocolDir = "protocol"
	CompiledDir = "compiled"

	ManifestFile = "compiled/.manifest.json"
	KBIndexFile  = "kb/index.md"
	KBMetaFile   = "kb/.meta.json"
)

// Layout resolves absolute paths inside a single workspace root.
type Layout struct{ Root string } // Root is the .hatch directory itself.

func (l Layout) path(parts ...string) string {
	return filepath.Join(append([]string{l.Root}, parts...)...)
}

func (l Layout) Charter() string         { return l.path(CharterFile) }
func (l Layout) Registry() string        { return l.path(RegistryFile) }
func (l Layout) Workflow() string        { return l.path(WorkflowFile) }
func (l Layout) Roles() string           { return l.path(RolesDir) }
func (l Layout) Context() string         { return l.path(ContextDir) }
func (l Layout) KB() string              { return l.path(KBDir) }
func (l Layout) Board() string           { return l.path(BoardDir) }
func (l Layout) Lane(name string) string { return l.path(BoardDir, name) }
func (l Layout) Ledger() string          { return l.path(LedgerDir) }
func (l Layout) Protocol() string        { return l.path(ProtocolDir) }
func (l Layout) Compiled() string        { return l.path(CompiledDir) }
func (l Layout) Manifest() string        { return l.path(ManifestFile) }
func (l Layout) KBIndex() string         { return l.path(KBIndexFile) }
func (l Layout) KBMeta() string          { return l.path(KBMetaFile) }

// RepoRoot is the directory that contains the .hatch directory.
func (l Layout) RepoRoot() string { return filepath.Dir(l.Root) }

// ErrNotFound indicates no .hatch workspace was located.
var ErrNotFound = errors.New("no .hatch workspace found (run `hatch init`)")

// Find walks up from start looking for a .hatch directory and returns its
// Layout. It stops at the filesystem root.
func Find(start string) (Layout, error) {
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

// At returns a Layout rooted at <dir>/.hatch without requiring it to exist.
func At(dir string) Layout {
	return Layout{Root: filepath.Join(dir, Dir)}
}
