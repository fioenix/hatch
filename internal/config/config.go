// Package config loads and validates the per-project registry and workflow
// definitions that configure a .hatch/ workspace.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/fioenix/hatch/internal/model"
	"github.com/fioenix/hatch/internal/paths"
)

// Workspace bundles the loaded configuration for a workspace.
type Workspace struct {
	Layout   paths.Layout
	Registry *model.Registry
	Workflow *model.Workflow
	// OutputRoot is where compiled surfaces (CLAUDE.md, .mcp.json, …) are
	// written. For a local .hatch it is the repo root (parent of .hatch); for
	// the global ~/.hatch it is the current working repo. Empty falls back to
	// Layout.RepoRoot().
	OutputRoot string
}

// Out returns the directory compiled outputs should be written to.
func (w *Workspace) Out() string {
	if w.OutputRoot != "" {
		return w.OutputRoot
	}
	return w.Layout.RepoRoot()
}

// LoadRegistry reads and parses registry.yaml.
func LoadRegistry(path string) (*model.Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r model.Registry
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &r, nil
}

// LoadWorkflow reads and parses workflow.yaml.
func LoadWorkflow(path string) (*model.Workflow, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var w model.Workflow
	if err := yaml.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &w, nil
}

// Load reads both config files for a workspace layout.
func Load(l paths.Layout) (*Workspace, error) {
	reg, err := LoadRegistry(l.Registry())
	if err != nil {
		return nil, err
	}
	wf, err := LoadWorkflow(l.Workflow())
	if err != nil {
		return nil, err
	}
	return &Workspace{Layout: l, Registry: reg, Workflow: wf}, nil
}
