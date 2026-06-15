package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/config"
	"github.com/fioenix/overclaud/hatch/internal/scaffold"
	"github.com/fioenix/overclaud/hatch/internal/store"
)

func TestBoardFlagsUnsafeTicketID(t *testing.T) {
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := config.Load(l)
	// A crafted ticket file: safe filename, but a traversal id in frontmatter.
	raw := "---\nid: \"../evil\"\nstatus: backlog\nrole: implementer\n---\nx\n"
	if err := os.WriteFile(filepath.Join(l.Lane("backlog"), "weird.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	var joined string
	for _, p := range Board(ws, store.NewBoard(l)) {
		joined += p.String() + "\n"
	}
	if !strings.Contains(joined, "unsafe ticket id") {
		t.Fatalf("expected unsafe-id problem, got:\n%s", joined)
	}
}
