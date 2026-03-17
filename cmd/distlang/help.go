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
	{Name: "helpers", Description: "Manage Distlang helper auth session and store access", Usage: "distlang helpers <login|store|whoami|logout>"},
	{Name: "run", Description: "Run the local V8 runtime", Usage: "distlang run <file> [--v8-port=N] [--set=all|handlerSet1|handlerSet2] [--port1=N] [--port2=N]"},
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
	commandHelpHelpersStore()
	fmt.Println()
	fmt.Println("---")
	commandHelpHelpersStoreObjectDB()
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
	fmt.Println("  - dist/v8/worker.js for single-worker apps")
	fmt.Println("  - dist/v8/handlerSet1/worker.js and dist/v8/handlerSet2/worker.js for simpleApp")
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
	fmt.Println("helpers - Manage Distlang helper auth session and store access")
	fmt.Println("Usage: distlang helpers <login|store|whoami|logout>")
	fmt.Println("Subcommands:")
	fmt.Println("  login    Start browser-based login against Distlang auth")
	fmt.Println("  store    Access authenticated helper store services")
	fmt.Println("  whoami   Show the current helper auth user")
	fmt.Println("  logout   Revoke local helper auth session")
	fmt.Println("Environment:")
	fmt.Println("  DISTLANG_AUTH_BASE_URL   Override auth service base URL (default: https://auth.distlang.com)")
	fmt.Println("  DISTLANG_STORE_BASE_URL  Override store API base URL (default: https://api.distlang.com)")
}

func commandHelpHelpersStore() {
	fmt.Println("helpers store - Access authenticated helper store services")
	fmt.Println("Usage: distlang helpers store <objectdb>")
	fmt.Println("Subcommands:")
	fmt.Println("  objectdb   Manage ObjectDB buckets, keys, and values")
	fmt.Println("Environment:")
	fmt.Println("  DISTLANG_STORE_BASE_URL  Override store API base URL (default: https://api.distlang.com)")
}

func commandHelpHelpersStoreObjectDB() {
	fmt.Println("helpers store objectdb - Manage authenticated ObjectDB data")
	fmt.Println("Usage: distlang helpers store objectdb <status|buckets|keys|put|get|head|delete>")
	fmt.Println("Subcommands:")
	fmt.Println("  status               Show ObjectDB service status for the logged-in user")
	fmt.Println("  buckets list         List buckets")
	fmt.Println("  buckets create NAME  Create a bucket")
	fmt.Println("  buckets exists NAME  Check whether a bucket exists")
	fmt.Println("  buckets delete NAME  Delete an empty bucket")
	fmt.Println("  keys list BUCKET     List keys in a bucket (keys only supports list)")
	fmt.Println("  put BUCKET KEY       Write a value with --file= or --text=")
	fmt.Println("  get BUCKET KEY       Read a value with --type=text|json|bytes")
	fmt.Println("  head BUCKET KEY      Show value metadata")
	fmt.Println("  delete BUCKET KEY    Delete a value")
}

func commandHelpHelpersStoreObjectDBStatus() {
	fmt.Println("helpers store objectdb status - Show ObjectDB service status")
	fmt.Println("Usage: distlang helpers store objectdb status")
	fmt.Println("Notes:")
	fmt.Println("  Uses the saved helper auth session and prints the effective store base URL")
}

func commandHelpHelpersStoreObjectDBBuckets() {
	fmt.Println("helpers store objectdb buckets - Manage ObjectDB buckets")
	fmt.Println("Usage: distlang helpers store objectdb buckets <list|create|exists|delete> [bucket]")
	fmt.Println("Examples:")
	fmt.Println("  distlang helpers store objectdb buckets list")
	fmt.Println("  distlang helpers store objectdb buckets create demo")
	fmt.Println("  distlang helpers store objectdb buckets exists demo")
	fmt.Println("  distlang helpers store objectdb buckets delete demo")
}

func commandHelpHelpersStoreObjectDBKeys() {
	fmt.Println("helpers store objectdb keys - List ObjectDB keys")
	fmt.Println("Usage: distlang helpers store objectdb keys list <bucket> [--prefix=value] [--limit=N] [--cursor=value]")
	fmt.Println("Notes:")
	fmt.Println("  Prints each key with metadata when available, followed by pagination status")
	fmt.Println("  Value writes and reads use `distlang helpers store objectdb put|get|head|delete`")
}

func commandHelpHelpersStoreObjectDBPut() {
	fmt.Println("helpers store objectdb put - Write an ObjectDB value")
	fmt.Println("Usage: distlang helpers store objectdb put <bucket> <key> [--file=path | --text=value] [--content-type=type]")
	fmt.Println("Notes:")
	fmt.Println("  Exactly one of --file or --text is required")
}

func commandHelpHelpersStoreObjectDBGet() {
	fmt.Println("helpers store objectdb get - Read an ObjectDB value")
	fmt.Println("Usage: distlang helpers store objectdb get <bucket> <key> [--type=text|json|bytes] [--output=path]")
	fmt.Println("Notes:")
	fmt.Println("  --type=text is the default; --type=bytes requires --output to avoid dumping binary to the terminal")
}

func commandHelpHelpersStoreObjectDBHead() {
	fmt.Println("helpers store objectdb head - Show ObjectDB value metadata")
	fmt.Println("Usage: distlang helpers store objectdb head <bucket> <key>")
}

func commandHelpHelpersStoreObjectDBDelete() {
	fmt.Println("helpers store objectdb delete - Delete an ObjectDB value")
	fmt.Println("Usage: distlang helpers store objectdb delete <bucket> <key>")
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
	fmt.Println("Usage: distlang run <file> [--v8-port=N] [--set=all|handlerSet1|handlerSet2] [--port1=N] [--port2=N]")
	fmt.Println("Options:")
	fmt.Println("  --v8-port=N     Port for local workerd (default: 5656, single-worker apps)")
	fmt.Println("  --set=...       Select simpleApp runtime set (default: all)")
	fmt.Println("  --port1=N       Port for handlerSet1 when running simpleApp (default: 5656)")
	fmt.Println("  --port2=N       Port for handlerSet2 when running simpleApp (default: 5657)")
	fmt.Println("Notes:")
	fmt.Println("  run builds the V8 backend first, then launches workerd instances.")
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
