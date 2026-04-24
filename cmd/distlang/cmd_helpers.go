package main

import (
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/auth"
)

func runHelpers(args []string) int {
	if len(args) == 0 {
		commandHelpHelpers()
		return 1
	}

	if args[0] == "-h" || args[0] == "--help" {
		commandHelpHelpers()
		return 0
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "auth":
		return runHelpersAuth(subArgs)
	case "login":
		return runHelpersLogin(subArgs)
	case "request":
		return runHelpersRequest(subArgs)
	case "store":
		return runHelpersStore(subArgs)
	case "whoami":
		return runHelpersWhoami(subArgs)
	case "logout":
		return runHelpersLogout(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown helpers subcommand: %s\n", subcommand)
		commandHelpHelpers()
		return 1
	}
}

func runHelpersLogin(args []string) int {
	if len(args) > 0 {
		if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
			commandHelpHelpersLogin()
			return 0
		}
		fmt.Fprintln(os.Stderr, "helpers login does not accept positional arguments")
		return 1
	}

	result, err := auth.Login()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers login failed: %v\n", err)
		return 1
	}

	fmt.Printf("Logged in as %s (%s)\n", result.User.Name, result.User.Email)
	return 0
}

func runHelpersWhoami(args []string) int {
	if len(args) > 0 {
		if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
			commandHelpHelpersWhoami()
			return 0
		}
		fmt.Fprintln(os.Stderr, "helpers whoami does not accept positional arguments")
		return 1
	}

	client := auth.NewClient(auth.ResolveBaseURL())
	session, err := client.EnsureSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers whoami failed: %v\n", err)
		return 1
	}

	identity, err := client.WhoAmI(session.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers whoami failed: %v\n", err)
		return 1
	}

	if identity.User.Name != "" {
		fmt.Printf("%s <%s>\n", identity.User.Name, identity.User.Email)
	} else {
		fmt.Println(identity.User.Email)
	}
	if identity.Token.ExpiresAt != "" {
		fmt.Printf("Token expires at: %s\n", identity.Token.ExpiresAt)
	}
	return 0
}

func runHelpersLogout(args []string) int {
	if len(args) > 0 {
		if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
			commandHelpHelpersLogout()
			return 0
		}
		fmt.Fprintln(os.Stderr, "helpers logout does not accept positional arguments")
		return 1
	}

	client := auth.NewClient(auth.ResolveBaseURL())
	if err := client.LogoutAndClear(); err != nil {
		fmt.Fprintf(os.Stderr, "helpers logout failed: %v\n", err)
		return 1
	}

	fmt.Println("Logged out")
	return 0
}
