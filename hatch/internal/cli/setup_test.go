package cli

import (
	"reflect"
	"testing"
)

func TestParseClientSelection(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"1,2", []string{"cc", "codex"}},
		{"cc codex", []string{"cc", "codex"}},
		{"3", []string{"agy"}},
		{"1, kiro", []string{"cc", "kiro"}},
		{"9", []string{"9"}}, // out of range → passthrough (rejected later)
		{" , ,", nil},        // separators only → nothing
		{"codex,,agy", []string{"codex", "agy"}},
	}
	for _, c := range cases {
		if got := parseClientSelection(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseClientSelection(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
