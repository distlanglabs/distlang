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

	results, errs := backend.BuildAll(filePath)

	hasError := len(errs) > 0
	for name, err := range errs {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}

	for _, res := range results {
		if err := artifacts.WriteAll(res.Artifacts); err != nil {
			fmt.Fprintf(os.Stderr, "%s: write artifacts: %v\n", res.Backend, err)
			hasError = true
		}
	}

	v8Out, err := backend.BuildV8(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cloudflare: package provider artifacts: %v\n", err)
		hasError = true
	} else {
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

	fmt.Printf("Build succeeded for %d backends\n", len(results))
	for _, res := range results {
		fmt.Printf("- %s: ", res.Backend)
		for i, art := range res.Artifacts {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(art.Path)
		}
		fmt.Println()
	}
	fmt.Println("- cloudflare: dist/cloudflare/worker.js, dist/cloudflare/wrangler.toml, dist/cloudflare/Makefile")

	return 0
}

func fileBase(filePath string) string {
	base := filepath.Base(filePath)
	return base[:len(base)-len(filepath.Ext(base))]
}
