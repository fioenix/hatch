package compile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Manifest records the hashes of SSOT inputs and the outputs they produced, so
// `hatch compile --check` can detect when compiled files are stale.
type Manifest struct {
	GeneratedAt string            `json:"generated_at"`
	Sources     map[string]string `json:"sources"` // path (rel to .hatch) → sha256
	Outputs     map[string]string `json:"outputs"` // path (rel to repo) → sha256
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// hashTree hashes every regular file under root, keyed by path relative to base.
func hashTree(root, base string, into map[string]string) {
	_ = filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(base, p)
		into[filepath.ToSlash(rel)] = hashBytes(raw)
		return nil
	})
}

// sourceHashes computes hashes of all SSOT inputs (charter, roles, context,
// registry, workflow), keyed relative to the .hatch root.
func sourceHashes(l paths.Layout) map[string]string {
	srcs := map[string]string{}
	for _, f := range []string{l.Charter(), l.WorkingAgreement(), l.Registry(), l.Workflow()} {
		if raw, err := os.ReadFile(f); err == nil {
			rel, _ := filepath.Rel(l.Root, f)
			srcs[filepath.ToSlash(rel)] = hashBytes(raw)
		}
	}
	hashTree(l.Roles(), l.Root, srcs)
	hashTree(l.Context(), l.Root, srcs)
	return srcs
}

// LoadManifest reads the manifest, returning an empty one if absent.
func LoadManifest(l paths.Layout) (*Manifest, error) {
	raw, err := os.ReadFile(l.Manifest())
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Sources: map[string]string{}, Outputs: map[string]string{}}, nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save writes the manifest to disk.
func (m *Manifest) Save(l paths.Layout) error {
	if err := os.MkdirAll(l.Compiled(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Manifest(), append(raw, '\n'), 0o644)
}

// StaleReason compares current SSOT + outputs against the manifest and returns
// a human-readable reason if anything drifted, or "" if up to date.
func StaleReason(l paths.Layout, repoRoot string) (string, error) {
	m, err := LoadManifest(l)
	if err != nil {
		return "", err
	}
	if m.GeneratedAt == "" {
		return "no manifest — compiled files have never been generated", nil
	}
	cur := sourceHashes(l)
	var changed []string
	for p, h := range cur {
		if m.Sources[p] != h {
			changed = append(changed, p)
		}
	}
	for p := range m.Sources {
		if _, ok := cur[p]; !ok {
			changed = append(changed, p+" (removed)")
		}
	}
	if len(changed) > 0 {
		sort.Strings(changed)
		return "SSOT changed since last compile: " + strings.Join(changed, ", "), nil
	}
	// Outputs edited by hand or missing?
	for rel, h := range m.Outputs {
		raw, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			return "compiled output missing: " + rel, nil
		}
		if hashBytes(raw) != h {
			return "compiled output edited by hand: " + rel, nil
		}
	}
	return "", nil
}
