package main

import (
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/passes"
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

	result, err := passes.Execute(filePath, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		return 1
	}

	fmt.Print(result.Emitted)
	return 0
}
