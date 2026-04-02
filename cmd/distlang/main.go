package main

import (
	"fmt"
	"os"
)

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
	if command == "-v" || command == "--version" {
		printVersion()
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
		case "target":
			commandHelpTarget()
		case "target-init", "targetinit":
			commandHelpTargetInit()
		case "deploy":
			commandHelpDeploy()
		case "helpers":
			commandHelpHelpers()
		case "helpers-store", "helpersstore":
			commandHelpHelpersStore()
		case "helpers-store-objectdb", "helpersstoreobjectdb":
			commandHelpHelpersStoreObjectDB()
		case "helpers-login", "helperslogin":
			commandHelpHelpersLogin()
		case "helpers-whoami", "helperswhoami":
			commandHelpHelpersWhoami()
		case "helpers-logout", "helperslogout":
			commandHelpHelpersLogout()
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
	case "target":
		os.Exit(runTarget(args))
	case "deploy":
		os.Exit(runDeploy(args))
	case "helpers":
		os.Exit(runHelpers(args))
	case "run":
		os.Exit(runRun(args))
	case "debug":
		os.Exit(runDebug(args))
	case "version":
		printVersion()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		usage()
		os.Exit(1)
	}
}
