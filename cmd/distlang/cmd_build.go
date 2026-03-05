package main

import (
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/platform"
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

	results, errs := platform.BuildAll(filePath)

	hasError := len(errs) > 0
	for p, err := range errs {
		fmt.Fprintf(os.Stderr, "%s: %v\n", p, err)
	}

	for _, res := range results {
		if err := artifacts.WriteAll(res.Artifacts); err != nil {
			fmt.Fprintf(os.Stderr, "%s: write artifacts: %v\n", res.Platform, err)
			hasError = true
		}
	}

	if hasError {
		fmt.Fprintln(os.Stderr, "Build failed")
		return 1
	}

	fmt.Printf("Build succeeded for %d platforms\n", len(results))
	for _, res := range results {
		fmt.Printf("- %s: ", res.Platform)
		for i, art := range res.Artifacts {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(art.Path)
		}
		fmt.Println()
	}

	return 0
}
