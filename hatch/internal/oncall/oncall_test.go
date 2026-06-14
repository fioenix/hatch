package oncall

import (
	"testing"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

func TestRotation(t *testing.T) {
	l := paths.At(t.TempDir())
	if Load(l).Now() != "" {
		t.Fatal("empty rotation should have no on-call")
	}
	r := Rotation{Order: []string{"a", "b", "c"}}
	if r.Now() != "a" {
		t.Fatalf("first on-call should be a, got %s", r.Now())
	}
	if r.Rotate() != "b" || r.Rotate() != "c" || r.Rotate() != "a" {
		t.Fatal("rotation wrap wrong")
	}
	if err := r.Save(l); err != nil {
		t.Fatal(err)
	}
	if Load(l).Now() != "a" {
		t.Fatalf("persisted current wrong: %s", Load(l).Now())
	}
}
