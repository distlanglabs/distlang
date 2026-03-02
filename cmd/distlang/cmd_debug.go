package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/distlanglabs/distlang/pkg/debug"
)

func runDebug(args []string) int {
	if len(args) >= 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpDebug()
		return 0
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "debug requires a target command and file path")
		fmt.Fprintln(os.Stderr, "Usage: distlang debug <build|run> <file> [--passes=parse,ir,emit]")
		return 1
	}

	target := args[0]
	passes := []string{"ir"}
	filePath := ""

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--passes=") {
			passes = parsePasses(arg)
			if len(passes) == 0 {
				fmt.Fprintln(os.Stderr, "no passes provided")
				return 1
			}
			continue
		}

		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "unknown debug flag: %s\n", arg)
			return 1
		}

		if filePath == "" {
			filePath = arg
		} else {
			fmt.Fprintln(os.Stderr, "debug accepts only one file path")
			return 1
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "debug requires a file path")
		return 1
	}

	execute := target == "run"
	if target != "build" && target != "run" {
		fmt.Fprintf(os.Stderr, "unknown debug target: %s\n", target)
		return 1
	}

	return debugFile(filePath, passes, execute)
}

func debugFile(filePath string, passes []string, execute bool) int {
	if err := debug.Run(filePath, passes, execute); err != nil {
		fmt.Fprintf(os.Stderr, "debug failed: %v\n", err)
		return 1
	}
	return 0
}
