package docs

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/scaffold"
)

func TestNewAndLintRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l, _, err := scaffold.Init(scaffold.Options{Dir: dir, Workflow: "scrum"})
	if err != nil {
		t.Fatal(err)
	}
	tmpls, err := Load(l)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tmpls["adr"]; !ok {
		t.Fatalf("default adr template missing; got %v", keys(tmpls))
	}
	// A freshly scaffolded ADR (sections intact) should lint clean.
	content, err := New(tmpls["adr"], "Use CSV streaming")
	if err != nil {
		t.Fatal(err)
	}
	dt, probs, err := Lint(content, tmpls)
	if err != nil {
		t.Fatal(err)
	}
	if dt != "adr" || len(probs) != 0 {
		t.Fatalf("fresh adr should be clean: type=%s probs=%v", dt, probs)
	}
	// Removing a required section should be flagged.
	broken := strings.Replace(string(content), "## Consequences", "## Misc", 1)
	_, probs2, _ := Lint([]byte(broken), tmpls)
	if len(probs2) == 0 {
		t.Fatal("expected a missing-section problem")
	}
}

func keys(m map[string]Template) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	return k
}
