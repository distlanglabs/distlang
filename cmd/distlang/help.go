package main

import "fmt"

type commandInfo struct {
	Name        string
	Description string
	Usage       string
}

var commands = []commandInfo{
	{Name: "build", Description: "Build backend artifacts and provider packages", Usage: "distlang build <file>"},
	{Name: "target", Description: "Manage target setup scaffolding", Usage: "distlang target <subcommand>"},
	{Name: "deploy", Description: "Deploy a backend through a provider", Usage: "distlang deploy <file> [--target=cloudflare]"},
	{Name: "helpers", Description: "Manage Distlang helper auth session", Usage: "distlang helpers <login|whoami|logout>"},
	{Name: "run", Description: "Run the local V8 runtime", Usage: "distlang run <file> [--v8-port=N]"},
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
	fmt.Println("Current work focuses on the JavaScript worker path and provider packaging.")
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
	commandHelpHelpers()
	fmt.Println()
	fmt.Println("---")
	commandHelpRun()
	fmt.Println()
	fmt.Println("---")
	commandHelpDebug()
}

func commandHelpBuild() {
	fmt.Println("build - Build backend artifacts and provider packages")
	fmt.Println("Usage: distlang build <file>")
	fmt.Println("Outputs:")
	fmt.Println("  - dist/v8/* backend artifacts")
	fmt.Println("  - dist/cloudflare/* provider package from V8 output")
}

func commandHelpDeploy() {
	fmt.Println("deploy - Deploy a backend through a provider")
	fmt.Println("Usage: distlang deploy <file> [--target=cloudflare]")
	fmt.Println("Options:")
	fmt.Println("  --target=cloudflare   Deploy target provider (default: cloudflare)")
	fmt.Println("Cloudflare credentials are loaded from shell env or <worker-dir>/targets/cloudflare/cloudflare.env")
}

func commandHelpTarget() {
	fmt.Println("target - Manage target setup scaffolding")
	fmt.Println("Usage: distlang target <subcommand>")
	fmt.Println("Subcommands:")
	fmt.Println("  init   Create target scaffolding for a project directory")
}

func commandHelpHelpers() {
	fmt.Println("helpers - Manage Distlang helper auth session")
	fmt.Println("Usage: distlang helpers <login|whoami|logout>")
	fmt.Println("Subcommands:")
	fmt.Println("  login    Start browser-based login against Distlang auth")
	fmt.Println("  whoami   Show the current helper auth user")
	fmt.Println("  logout   Revoke local helper auth session")
	fmt.Println("Environment:")
	fmt.Println("  DISTLANG_AUTH_BASE_URL   Override auth service base URL (default: https://auth.distlang.com)")
}

func commandHelpHelpersLogin() {
	fmt.Println("helpers login - Start browser-based helper login")
	fmt.Println("Usage: distlang helpers login")
	fmt.Println("Notes:")
	fmt.Println("  Opens the browser to Google login, then listens on http://127.0.0.1:8976/callback")
	fmt.Println("  Stores the resulting session in your user config directory")
}

func commandHelpHelpersWhoami() {
	fmt.Println("helpers whoami - Show current helper auth user")
	fmt.Println("Usage: distlang helpers whoami")
	fmt.Println("Notes:")
	fmt.Println("  Refreshes the local session automatically when needed")
}

func commandHelpHelpersLogout() {
	fmt.Println("helpers logout - Clear current helper auth session")
	fmt.Println("Usage: distlang helpers logout")
	fmt.Println("Notes:")
	fmt.Println("  Revokes the remote refresh token when available and clears the local session")
}

func commandHelpTargetInit() {
	fmt.Println("target init - Create target scaffolding")
	fmt.Println("Usage: distlang target init [--target=cloudflare[,cloudflare...]] [--path=.]")
	fmt.Println("Options:")
	fmt.Println("  --target=...   Comma-separated targets to initialize (default: cloudflare)")
	fmt.Println("  --path=...     Project directory where targets/ will be created (default: current directory)")
}

func commandHelpRun() {
	fmt.Println("run - Run the local V8 runtime")
	fmt.Println("Usage: distlang run <file> [--v8-port=N]")
	fmt.Println("Options:")
	fmt.Println("  --v8-port=N     Port for local workerd (default: 5656)")
	fmt.Println("Notes:")
	fmt.Println("  run builds the V8 backend first, then launches workerd.")
}

func commandHelpDebug() {
	fmt.Println("debug - Inspect compiler passes for build or run")
	fmt.Println("Usage: distlang debug <build|run> <file> [--passes=parse,ir,emit]")
	fmt.Println("Options:")
	fmt.Println("  --passes=parse,ir,emit   Comma-separated passes to print (default: ir)")
	fmt.Println("    parse  - show transformed backend-ready JS")
	fmt.Println("    ir     - print normalized IR as JSON")
	fmt.Println("    emit   - emitted JS (same as parse for now)")
	fmt.Println("  debug run no longer executes in-process; use distlang run instead")
}
