package main

import "fmt"

type commandInfo struct {
	Name        string
	Description string
	Usage       string
}

var commands = []commandInfo{
	{Name: "build", Description: "Build platform artifacts (goja, cloudflare)", Usage: "distlang build <file>"},
	{Name: "target", Description: "Manage target setup scaffolding", Usage: "distlang target <subcommand>"},
	{Name: "deploy", Description: "Deploy worker to a target platform", Usage: "distlang deploy <file> [--target=cloudflare]"},
	{Name: "run", Description: "Serve worker fetch via Goja", Usage: "distlang run <file> [--port=N]"},
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
	fmt.Println("Command Help:")
	fmt.Println("---")
	commandHelpBuild()
	fmt.Println()
	fmt.Println("---")
	commandHelpTarget()
	fmt.Println()
	fmt.Println("---")
	commandHelpDeploy()
	fmt.Println()
	fmt.Println("---")
	commandHelpRun()
	fmt.Println()
	fmt.Println("---")
	commandHelpDebug()
}

func commandHelpBuild() {
	fmt.Println("build - Build platform artifacts (goja, cloudflare)")
	fmt.Println("Usage: distlang build <file>")
}

func commandHelpDeploy() {
	fmt.Println("deploy - Deploy worker to a target platform")
	fmt.Println("Usage: distlang deploy <file> [--target=cloudflare]")
	fmt.Println("Options:")
	fmt.Println("  --target=cloudflare   Deploy target platform (default: cloudflare)")
	fmt.Println("Cloudflare credentials are loaded from shell env or <worker-dir>/targets/cloudflare/cloudflare.env")
}

func commandHelpTarget() {
	fmt.Println("target - Manage target setup scaffolding")
	fmt.Println("Usage: distlang target <subcommand>")
	fmt.Println("Subcommands:")
	fmt.Println("  init   Create target scaffolding for a project directory")
}

func commandHelpTargetInit() {
	fmt.Println("target init - Create target scaffolding")
	fmt.Println("Usage: distlang target init [--target=cloudflare[,cloudflare...]] [--path=.]")
	fmt.Println("Options:")
	fmt.Println("  --target=...   Comma-separated targets to initialize (default: cloudflare)")
	fmt.Println("  --path=...     Project directory where targets/ will be created (default: current directory)")
}

func commandHelpRun() {
	fmt.Println("run - Serve worker fetch via Goja")
	fmt.Println("Usage: distlang run <file> [--port=N]")
	fmt.Println("Options:")
	fmt.Println("  --port=N   Port to listen on (default: 5656)")
}

func commandHelpDebug() {
	fmt.Println("debug - Inspect compiler passes for build or run")
	fmt.Println("Usage: distlang debug <build|run> <file> [--passes=parse,ir,emit]")
	fmt.Println("Options:")
	fmt.Println("  --passes=parse,ir,emit   Comma-separated passes to print (default: ir)")
	fmt.Println("    parse  - show transformed Goja-ready JS")
	fmt.Println("    ir     - print normalized IR (Goja-friendly) as JSON")
	fmt.Println("    emit   - emitted JS (same as parse for now)")
}
