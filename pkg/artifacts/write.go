package artifacts

import (
	"os"
	"path/filepath"
)

// Artifact represents a file to be written.
type Artifact struct {
	Path    string
	Content []byte
}

// WriteAll writes the provided artifacts to disk, creating parent directories as needed.
func WriteAll(items []Artifact) error {
	for _, a := range items {
		dir := filepath.Dir(a.Path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(a.Path, a.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}
