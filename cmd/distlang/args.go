package main

import (
	"fmt"
	"os"
	"strings"
)

func singlePathArg(args []string, command string) (string, error) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "%s requires exactly one file path\n", command)
		fmt.Fprintf(os.Stderr, "Usage: distlang %s <file>\n", command)
		return "", fmt.Errorf("missing path")
	}
	return args[0], nil
}

func parsePasses(flag string) []string {
	parts := strings.TrimPrefix(flag, "--passes=")
	if parts == "" {
		return nil
	}
	items := strings.Split(parts, ",")
	var cleaned []string
	for _, p := range items {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}
