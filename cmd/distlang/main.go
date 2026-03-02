package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/distlanglabs/distlang/pkg/debug"
	"github.com/distlanglabs/distlang/pkg/parser"
	"github.com/distlanglabs/distlang/pkg/runtime"
)

type commandInfo struct {
	Name        string
	Description string
	Usage       string
}

var commands = []commandInfo{
	{Name: "build", Description: "Read a source file and print its contents (POC)", Usage: "distlang build <file>"},
	{Name: "run", Description: "Execute a JS file with goja (POC)", Usage: "distlang run <file>"},
	{Name: "debug", Description: "Inspect compiler passes for build or run", Usage: "distlang debug <build|run> <file> [--passes=parse,ir,emit]"},
	{Name: "help", Description: "Show help for distlang", Usage: "distlang help"},
}

var globalFlags = []string{
	"-h, --help           Show help for distlang",
	"--full-help          Show global and per-command help",
}

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
	for _, c := range commands {
		fmt.Printf("  %-14s %s\n", c.Name, c.Description)
	}
	fmt.Println()
	fmt.Println("Flags:")
	for _, f := range globalFlags {
		fmt.Printf("  %s\n", f)
	}
	fmt.Println()
	fmt.Println("Tip: run 'distlang <command> --help' for command-specific options. Use --full-help to see everything.")
}

func fullHelp() {
	usage()
	fmt.Println()
	commandHelpBuild()
	fmt.Println()
	commandHelpRun()
	fmt.Println()
	commandHelpDebug()
}

func runBuild(args []string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpBuild()
		return 0
	}

	filePath, err := singlePathArg(args, "build")
	if err != nil {
		return 1
	}

	contents, err := parser.ParseFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		return 1
	}

	fmt.Print(contents)
	return 0
}

func commandHelpBuild() {
	fmt.Println("build - Read a source file and print its contents (POC)")
	fmt.Println("Usage: distlang build <file>")
}

func runRun(args []string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpRun()
		return 0
	}

	filePath, err := singlePathArg(args, "run")
	if err != nil {
		return 1
	}

	source, err := parser.ParseFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	engine := runtime.NewDefaultEngine()
	if err := engine.RunScript(filePath, source); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	return 0
}

func commandHelpRun() {
	fmt.Println("run - Execute a JS file with goja (POC)")
	fmt.Println("Usage: distlang run <file>")
}

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

func commandHelpDebug() {
	fmt.Println("debug - Inspect compiler passes for build or run")
	fmt.Println("Usage: distlang debug <build|run> <file> [--passes=parse,ir,emit]")
	fmt.Println("Options:")
	fmt.Println("  --passes=parse,ir,emit   Comma-separated passes to print (default: ir)")
}

func debugFile(filePath string, passes []string, execute bool) int {
	if err := debug.Run(filePath, passes, execute); err != nil {
		fmt.Fprintf(os.Stderr, "debug failed: %v\n", err)
		return 1
	}
	return 0
}

func singlePathArg(args []string, command string) (string, error) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "%s requires exactly one file path\n", command)
		fmt.Fprintf(os.Stderr, "Usage: distlang %s <file>\n", command)
		return "", fmt.Errorf("missing path")
	}
	return args[0], nil
}

func parsePasses(flag string) []string {
	parts := strings.TrimPrefix(flag, "--passes=")
	if parts == "" {
		return nil
	}
	items := strings.Split(parts, ",")
	var cleaned []string
	for _, p := range items {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("distlang - portable distributed app framework (POC)")
		fmt.Println("Run `distlang -h` for usage.")
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	// Global help flags.
	if command == "-h" || command == "--help" {
		usage()
		return
	}
	if command == "--full-help" {
		fullHelp()
		return
	}

	// Allow `distlang help` and `distlang help <command>`.
	if command == "help" {
		if len(args) == 0 {
			usage()
			return
		}
		switch args[0] {
		case "build":
			commandHelpBuild()
		case "run":
			commandHelpRun()
		case "debug":
			commandHelpDebug()
		case "--full-help", "--fullhelp", "full":
			fullHelp()
		default:
			fmt.Fprintf(os.Stderr, "unknown help topic: %s\n\n", args[0])
			usage()
		}
		return
	}

	switch command {
	case "build":
		os.Exit(runBuild(args))
	case "run":
		os.Exit(runRun(args))
	case "debug":
		os.Exit(runDebug(args))
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		usage()
		os.Exit(1)
	}
}
