package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/backend"
	cloudflareprovider "github.com/distlanglabs/distlang/pkg/provider/cloudflare"
)

func runBuild(args []string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpBuild()
		return 0
	}

	filePath, err := singlePathArg(args, "build")
	if err != nil {
		return 1
	}

	v8Out, err := backend.BuildV8(filePath)
	hasError := false
	if err != nil {
		fmt.Fprintf(os.Stderr, "v8: build backend artifacts: %v\n", err)
		hasError = true
	} else {
		if err := artifacts.WriteAll(v8Out.Artifacts); err != nil {
			fmt.Fprintf(os.Stderr, "v8: write artifacts: %v\n", err)
			hasError = true
		}

		projectName := fileBase(filePath)
		packaged, err := cloudflareprovider.Package(v8Out, cloudflareprovider.Context{ProjectName: projectName})
		if err != nil {
			fmt.Fprintf(os.Stderr, "cloudflare: package provider artifacts: %v\n", err)
			hasError = true
		} else if err := artifacts.WriteAll(packaged); err != nil {
			fmt.Fprintf(os.Stderr, "cloudflare: write artifacts: %v\n", err)
			hasError = true
		}
	}

	if hasError {
		fmt.Fprintln(os.Stderr, "Build failed")
		return 1
	}

	fmt.Println("Build succeeded")
	fmt.Println("- v8: dist/v8/worker.js")
	fmt.Println("- cloudflare: dist/cloudflare/worker.js, dist/cloudflare/wrangler.toml, dist/cloudflare/Makefile")

	return 0
}

func fileBase(filePath string) string {
	base := filepath.Base(filePath)
	return base[:len(base)-len(filepath.Ext(base))]
}
