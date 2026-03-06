package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runTarget(args []string) int {
	if len(args) == 0 {
		commandHelpTarget()
		return 1
	}

	if args[0] == "-h" || args[0] == "--help" {
		commandHelpTarget()
		return 0
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "init":
		return runTargetInit(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown target subcommand: %s\n", subcommand)
		commandHelpTarget()
		return 1
	}
}

func runTargetInit(args []string) int {
	if len(args) >= 1 && (args[0] == "-h" || args[0] == "--help") {
		commandHelpTargetInit()
		return 0
	}

	path := "."
	targets := []string{}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--path=") {
			path = strings.TrimSpace(strings.TrimPrefix(arg, "--path="))
			continue
		}
		if strings.HasPrefix(arg, "--target=") {
			raw := strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
			for _, t := range strings.Split(raw, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					targets = append(targets, t)
				}
			}
			continue
		}

		fmt.Fprintf(os.Stderr, "unknown target init flag: %s\n", arg)
		return 1
	}

	if len(targets) == 0 {
		targets = []string{"cloudflare"}
	}

	for _, target := range targets {
		switch target {
		case "cloudflare":
			if err := initCloudflareTarget(path); err != nil {
				fmt.Fprintf(os.Stderr, "target init failed: %v\n", err)
				return 1
			}
		default:
			fmt.Fprintf(os.Stderr, "unsupported target for init: %s\n", target)
			return 1
		}
	}

	return 0
}

func initCloudflareTarget(basePath string) error {
	targetDir := filepath.Join(basePath, "targets", "cloudflare")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	envExamplePath := filepath.Join(targetDir, "cloudflare.env.example")
	envPath := filepath.Join(targetDir, "cloudflare.env")

	if err := writeFileIfMissing(envExamplePath, []byte("# Copy to cloudflare.env and fill values.\nCLOUDFLARE_API_TOKEN=\nCLOUDFLARE_ACCOUNT_ID=\n")); err != nil {
		return err
	}
	if err := writeFileIfMissing(envPath, []byte("# Local Cloudflare deploy credentials (do not commit).\nCLOUDFLARE_API_TOKEN=\nCLOUDFLARE_ACCOUNT_ID=\n")); err != nil {
		return err
	}

	fmt.Printf("Initialized cloudflare target in %s\n", targetDir)
	return nil
}

func writeFileIfMissing(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Skipped existing file %s\n", path)
		return nil
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", path)
	return nil
}
