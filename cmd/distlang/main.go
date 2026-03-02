package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/parser"
	gojaengine "github.com/distlanglabs/distlang/pkg/runtime/goja"
)

func usage() {
	fmt.Println("distlang - portable distributed app framework (POC)")
	fmt.Println()
	fmt.Println("Distlang is a capability-based framework for building portable serverless apps.")
	fmt.Println("Phase 0 focuses on local development with http, kv, and log capabilities.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  distlang <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build <file>   Read a source file and print its contents (POC)")
	fmt.Println("  run <file>     Execute a JS file with goja (POC)")
	fmt.Println("  help           Show help for distlang")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h, --help     Show help for distlang")
	fmt.Println("  --debug-ir     With 'run', print normalized IR to stderr")
}

func runBuild(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "build requires exactly one file path")
		fmt.Fprintln(os.Stderr, "Usage: distlang build <file>")
		return 1
	}

	contents, err := parser.ParseFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		return 1
	}

	fmt.Print(contents)
	return 0
}

func runRun(args []string) int {
	debugIR := false
	filePath := ""

	for _, arg := range args {
		switch arg {
		case "--debug-ir":
			debugIR = true
		default:
			if filePath == "" {
				filePath = arg
			} else {
				fmt.Fprintln(os.Stderr, "run accepts at most one file path")
				fmt.Fprintln(os.Stderr, "Usage: distlang run [--debug-ir] <file>")
				return 1
			}
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "run requires a file path")
		fmt.Fprintln(os.Stderr, "Usage: distlang run [--debug-ir] <file>")
		return 1
	}

	source, err := parser.ParseFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	if debugIR {
		ir, err := gojaengine.BuildIR(filePath, source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug-ir failed: %v\n", err)
			return 1
		}
		data, err := json.MarshalIndent(ir, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "debug-ir marshal failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(os.Stderr, string(data))
	}

	engine := gojaengine.NewEngine()
	if err := engine.RunScript(filePath, source); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	return 0
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("distlang - portable distributed app framework (POC)")
		fmt.Println("Run `distlang -h` for usage.")
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "-h", "--help", "help":
		usage()
		return
	case "build":
		os.Exit(runBuild(args))
	case "run":
		os.Exit(runRun(args))
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		usage()
		os.Exit(1)
	}
}
