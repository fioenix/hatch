// Package mdfront parses and serializes Markdown files that carry a YAML
// frontmatter block delimited by `---` lines, as used by tickets, KB entries
// and role files throughout a .hatch/ workspace.
package mdfront

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const fence = "---"

// Doc is a parsed frontmatter document: the raw YAML node plus the Markdown
// body that follows it.
type Doc struct {
	Meta yaml.Node // frontmatter as a YAML node (zero value if none)
	Body string    // everything after the closing fence
}

// Parse splits raw bytes into frontmatter and body. A document without a
// leading `---` fence is treated as all-body with empty Meta.
func Parse(raw []byte) (*Doc, error) {
	s := string(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n")))
	if !strings.HasPrefix(s, fence+"\n") && s != fence {
		return &Doc{Body: s}, nil
	}
	rest := strings.TrimPrefix(s, fence+"\n")
	end := strings.Index(rest, "\n"+fence)
	if end < 0 {
		return nil, fmt.Errorf("frontmatter opened with %q but never closed", fence)
	}
	front := rest[:end]
	body := rest[end+len("\n"+fence):]
	// Drop the rest of the fence line and any blank lines Encode inserts for
	// readability, so Decode(Encode(v, body)) == body.
	if nl := strings.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	} else {
		body = ""
	}
	body = strings.TrimLeft(body, "\n")

	d := &Doc{Body: body}
	if strings.TrimSpace(front) != "" {
		if err := yaml.Unmarshal([]byte(front), &d.Meta); err != nil {
			return nil, fmt.Errorf("invalid frontmatter YAML: %w", err)
		}
	}
	return d, nil
}

// Decode parses raw bytes and unmarshals the frontmatter into v.
func Decode(raw []byte, v any) (body string, err error) {
	d, err := Parse(raw)
	if err != nil {
		return "", err
	}
	if d.Meta.Kind != 0 {
		if err := d.Meta.Decode(v); err != nil {
			return "", fmt.Errorf("decode frontmatter: %w", err)
		}
	}
	return d.Body, nil
}

// Encode renders frontmatter v plus a Markdown body into a single document.
func Encode(v any, body string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(fence + "\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encode frontmatter: %w", err)
	}
	_ = enc.Close()
	buf.WriteString(fence + "\n")
	if body != "" {
		if !strings.HasPrefix(body, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			buf.WriteString("\n")
		}
	}
	return buf.Bytes(), nil
}
