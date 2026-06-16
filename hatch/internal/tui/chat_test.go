package tui

import (
	"strings"
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/config"
)

func TestRenderBodyMarkdown(t *testing.T) {
	c := &chat{ws: &config.Workspace{}}
	c.w, c.h = 100, 40
	c.layout() // builds the glamour renderer
	if c.md == nil {
		t.Fatal("markdown renderer not built")
	}
	body := "## Heading\n\n```go\nfunc Reverse(s string) string { return s }\n```\n\n- **bold** item"
	out := c.renderBody(body)
	// Markdown must be RENDERED, not shown raw: the ``` fence + ## must be gone.
	if strings.Contains(out, "```") {
		t.Errorf("code fence not rendered (raw markdown leaked):\n%s", out)
	}
	if strings.Contains(out, "## Heading") {
		t.Errorf("heading not rendered (raw markdown leaked):\n%s", out)
	}
	// Code content should survive.
	if !strings.Contains(out, "Reverse") {
		t.Errorf("code content missing:\n%s", out)
	}
}

func TestTrunc(t *testing.T) {
	if got := trunc("dogfood-verify-agy-mcp-path", 10); len([]rune(got)) != 10 {
		t.Errorf("trunc len = %d, want 10 (%q)", len([]rune(got)), got)
	}
	if got := trunc("short", 10); got != "short" {
		t.Errorf("trunc shortened a fitting string: %q", got)
	}
}
