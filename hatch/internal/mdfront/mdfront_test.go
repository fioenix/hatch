package mdfront

import "testing"

type meta struct {
	ID   string   `yaml:"id"`
	Tags []string `yaml:"tags,omitempty"`
}

func TestRoundTrip(t *testing.T) {
	m := meta{ID: "T-1", Tags: []string{"a", "b"}}
	raw, err := Encode(m, "## Body\nhello\n")
	if err != nil {
		t.Fatal(err)
	}
	var got meta
	body, err := Decode(raw, &got)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "T-1" || len(got.Tags) != 2 {
		t.Fatalf("meta round-trip failed: %+v", got)
	}
	if body != "## Body\nhello\n" {
		t.Fatalf("body mismatch: %q", body)
	}
}

func TestNoFrontmatter(t *testing.T) {
	d, err := Parse([]byte("just body\n"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Meta.Kind != 0 {
		t.Fatal("expected empty meta")
	}
	if d.Body != "just body\n" {
		t.Fatalf("body: %q", d.Body)
	}
}

func TestUnterminated(t *testing.T) {
	if _, err := Parse([]byte("---\nid: x\nno closing fence\n")); err == nil {
		t.Fatal("expected error for unterminated frontmatter")
	}
}
