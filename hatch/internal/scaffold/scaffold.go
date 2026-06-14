// Package scaffold creates a fresh .hatch/ workspace from embedded templates.
package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

//go:embed all:templates
var templates embed.FS

// WorkflowTemplates lists the built-in workflow.yaml variants `hatch init`
// can install (see docs/05-workflow.md).
var WorkflowTemplates = []string{
	"scrum", "kanban", "spec-first", "lite",
	"dual-track", "shape-up", "stage-gate", "incident",
}

// Options configure an init run.
type Options struct {
	Dir      string // directory to create .hatch under
	Workflow string // one of WorkflowTemplates
	Force    bool   // overwrite an existing .hatch
}

// Init writes a new .hatch workspace and returns its layout.
func Init(opt Options) (paths.Layout, []string, error) {
	if opt.Workflow == "" {
		opt.Workflow = "scrum"
	}
	if !validWorkflow(opt.Workflow) {
		return paths.Layout{}, nil, fmt.Errorf("unknown workflow template %q (choose: %s)",
			opt.Workflow, strings.Join(WorkflowTemplates, ", "))
	}
	l := paths.At(opt.Dir)
	if _, err := os.Stat(l.Root); err == nil && !opt.Force {
		return paths.Layout{}, nil, fmt.Errorf("%s already exists (use --force to overwrite)", l.Root)
	}

	var written []string
	// Copy the shared base tree (everything except the workflow variants dir).
	base, _ := fs.Sub(templates, "templates/base")
	err := fs.WalkDir(base, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := base.Open(p)
		if err != nil {
			return err
		}
		defer data.Close()
		content, err := fs.ReadFile(base, p)
		if err != nil {
			return err
		}
		// .keep files create empty directories only.
		dest := filepath.Join(l.Root, p)
		if filepath.Base(p) == ".keep" {
			return os.MkdirAll(filepath.Dir(dest), 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, content, 0o644); err != nil {
			return err
		}
		written = append(written, dest)
		return nil
	})
	if err != nil {
		return paths.Layout{}, nil, err
	}

	// Install the selected workflow.yaml.
	wf, err := templates.ReadFile("templates/workflows/" + opt.Workflow + ".yaml")
	if err != nil {
		return paths.Layout{}, nil, err
	}
	if err := os.WriteFile(l.Workflow(), wf, 0o644); err != nil {
		return paths.Layout{}, nil, err
	}
	written = append(written, l.Workflow())

	return l, written, nil
}

func validWorkflow(name string) bool {
	for _, w := range WorkflowTemplates {
		if w == name {
			return true
		}
	}
	return false
}
