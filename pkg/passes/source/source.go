package source

import (
	"fmt"
	"os"
)

// ReadFile returns the contents of the provided path as a string.
func ReadFile(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}
	return string(contents), nil
}
