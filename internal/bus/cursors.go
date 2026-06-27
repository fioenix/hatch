package bus

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (b *Bus) loadCursors() (map[string]string, error) {
	raw, err := os.ReadFile(b.cursorsPath())
	if err != nil {
		return map[string]string{}, nil
	}
	m := map[string]string{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]string{}, nil
	}
	return m, nil
}

func (b *Bus) saveCursors(m map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(b.cursorsPath()), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.cursorsPath(), append(raw, '\n'), 0o644)
}
