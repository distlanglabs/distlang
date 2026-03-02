package parser

import (
	"fmt"
	"os"
)

func ParseFile(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}

	return string(contents), nil
}
