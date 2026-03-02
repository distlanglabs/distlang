package main

import (
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/passes"
	"github.com/distlanglabs/distlang/pkg/runtime"
	runtimetypes "github.com/distlanglabs/distlang/pkg/runtime/types"
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
	resp, err := engine.RunWorker(filePath, result.Emitted, runtimetypes.Request{
		URL:    "http://localhost/",
		Method: "GET",
		Headers: map[string]string{
			"host": "localhost",
		},
	})
	if err != nil {
		// Fallback to plain script execution for non-worker scripts.
		if err := engine.RunScript(filePath, result.Emitted); err != nil {
			fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Printf("status: %d\n%s", resp.Status, resp.Body)
	return 0
}
