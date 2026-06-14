// Package docs implements the document template/spec system: agents scaffold
// PRDs, design docs, ADRs, postmortems… from per-project templates and can
// lint a document against its declared spec. Templates live in
// .hatch/templates/docs/ and are user-editable (see docs/16).
package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/mdfront"
	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Template is a document type's spec + scaffold body.
type Template struct {
	Type                string   `yaml:"doc-type"`
	Framework           string   `yaml:"framework"`
	RequiredFrontmatter []string `yaml:"required-frontmatter"`
	RequiredSections    []string `yaml:"required-sections"`
	Body                string   `yaml:"-"`
}

// Load reads all templates from .hatch/templates/docs/.
func Load(l paths.Layout) (map[string]Template, error) {
	dir := l.DocTemplates()
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Template{}, nil
		}
		return nil, err
	}
	out := map[string]Template{}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var t Template
		body, err := mdfront.Decode(raw, &t)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		t.Body = body
		if t.Type == "" {
			t.Type = strings.TrimSuffix(e.Name(), ".md")
		}
		out[t.Type] = t
	}
	return out, nil
}

// Types lists available doc types, sorted.
func Types(l paths.Layout) ([]Template, error) {
	m, err := Load(l)
	if err != nil {
		return nil, err
	}
	out := make([]Template, 0, len(m))
	for _, t := range m {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out, nil
}

// docMeta is the frontmatter written into a generated document.
type docMeta struct {
	DocType string `yaml:"doc-type"`
	ID      string `yaml:"id,omitempty"`
	Title   string `yaml:"title"`
	Status  string `yaml:"status,omitempty"`
	Date    string `yaml:"date,omitempty"`
}

// New renders a new document from a template, substituting {{title}} and
// seeding required frontmatter.
func New(t Template, title string) ([]byte, error) {
	meta := docMeta{DocType: t.Type, Title: title}
	for _, f := range t.RequiredFrontmatter {
		switch f {
		case "date":
			meta.Date = time.Now().Format("2006-01-02")
		case "status":
			meta.Status = "draft"
		case "id":
			meta.ID = slug(title)
		}
	}
	body := strings.ReplaceAll(t.Body, "{{title}}", title)
	return mdfront.Encode(meta, body)
}

// Lint checks a document against the template for its declared doc-type.
func Lint(raw []byte, templates map[string]Template) (docType string, problems []string, err error) {
	var meta docMeta
	body, err := mdfront.Decode(raw, &meta)
	if err != nil {
		return "", nil, err
	}
	if meta.DocType == "" {
		return "", []string{"missing `doc-type` in frontmatter"}, nil
	}
	t, ok := templates[meta.DocType]
	if !ok {
		return meta.DocType, []string{"unknown doc-type " + meta.DocType}, nil
	}
	// frontmatter presence
	have := map[string]bool{"title": meta.Title != "", "date": meta.Date != "", "status": meta.Status != "", "id": meta.ID != "", "doc-type": true}
	for _, f := range t.RequiredFrontmatter {
		if !have[f] {
			problems = append(problems, "thiếu frontmatter `"+f+"`")
		}
	}
	// required sections (## Heading)
	for _, s := range t.RequiredSections {
		if !hasSection(body, s) {
			problems = append(problems, "thiếu mục `## "+s+"`")
		}
	}
	return meta.DocType, problems, nil
}

func hasSection(body, section string) bool {
	want := strings.ToLower("## " + section)
	for _, ln := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(ln)), want) {
			return true
		}
	}
	return false
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// Slug exposes the slugifier for filename construction.
func Slug(s string) string { return slug(s) }
