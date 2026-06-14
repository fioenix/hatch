package model

// KB entry types (docs/09-knowledge-base.md).
const (
	KBDecision = "decision"
	KBDomain   = "domain"
	KBLearning = "learning"
)

// KBEntry is a unit of shared knowledge: a short Markdown file with frontmatter
// that every agent can read and write.
type KBEntry struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Title   string   `yaml:"title"`
	Tags    []string `yaml:"tags,omitempty"`
	Related []string `yaml:"related,omitempty"`
	Author  string   `yaml:"author,omitempty"`
	Created string   `yaml:"created,omitempty"`
	Status  string   `yaml:"status,omitempty"` // for decisions: proposed|accepted|superseded

	Body string `yaml:"-"`
	Path string `yaml:"-"` // path relative to kb/, set by the store
}

// Subdir returns the kb/ subdirectory an entry type lives in.
func KBSubdir(typ string) string {
	switch typ {
	case KBDecision:
		return "decisions"
	case KBDomain:
		return "domain"
	default:
		return "learnings"
	}
}
