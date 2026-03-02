package main

import (
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/parser"
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
	fmt.Println("  help           Show help for distlang")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h, --help     Show help for distlang")
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
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		usage()
		os.Exit(1)
	}
}
