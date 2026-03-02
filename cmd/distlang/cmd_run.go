package main

import (
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/passes"
	"github.com/distlanglabs/distlang/pkg/runtime"
)

func runRun(args []string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpRun()
		return 0
	}

	filePath, err := singlePathArg(args, "run")
	if err != nil {
		return 1
	}

	result, err := passes.Execute(filePath, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	engine := runtime.NewDefaultEngine()
	if err := engine.RunScript(filePath, result.Emitted); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	return 0
}
