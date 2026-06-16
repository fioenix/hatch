package cli

import "testing"

func TestEditTargetPath(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"claude", `{"tool_name":"Edit","tool_input":{"file_path":"/r/.hatch/charter.md"}}`, "/r/.hatch/charter.md"},
		{"agy", `{"toolCall":{"name":"write_to_file","args":{"TargetFile":"/r/x.go"}}}`, "/r/x.go"},
		{"no-file-tool", `{"tool_name":"Bash","tool_input":{"command":"ls"}}`, ""},
		{"empty", ``, ""},
		{"garbage", `not json`, ""},
	}
	for _, c := range cases {
		if got := editTargetPath([]byte(c.in)); got != c.want {
			t.Errorf("%s: editTargetPath = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestProtectedMatch(t *testing.T) {
	cases := []struct {
		glob, rel string
		want      bool
	}{
		{".hatch/charter.md", ".hatch/charter.md", true},
		{".hatch/registry.yaml", ".hatch/charter.md", false},
		{".hatch/*.yaml", ".hatch/registry.yaml", true},
		{".hatch/", ".hatch/charter.md", true},   // dir prefix
		{".hatch/**", ".hatch/roles/x.md", true}, // dir prefix (**)
		{".hatch/charter.md", "README.md", false},
	}
	for _, c := range cases {
		if got := protectedMatch(c.glob, c.rel); got != c.want {
			t.Errorf("protectedMatch(%q,%q) = %v, want %v", c.glob, c.rel, got, c.want)
		}
	}
}
