package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/distlanglabs/distlang/pkg/auth"
	storeapi "github.com/distlanglabs/distlang/pkg/store"
)

func runHelpersAuth(args []string) int {
	if len(args) == 0 {
		commandHelpHelpersAuth()
		return 1
	}

	if args[0] == "-h" || args[0] == "--help" {
		commandHelpHelpersAuth()
		return 0
	}

	switch args[0] {
	case "status":
		return runHelpersAuthStatus(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown helpers auth subcommand: %s\n", args[0])
		commandHelpHelpersAuth()
		return 1
	}
}

func runHelpersAuthStatus(args []string) int {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "-h", "--help":
			commandHelpHelpersAuthStatus()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "unknown helpers auth status flag: %s\n", arg)
			return 1
		}
	}

	client := auth.NewClient(auth.ResolveBaseURL())
	session, err := client.EnsureSession()
	if err != nil {
		if errors.Is(err, auth.ErrNotLoggedIn) {
			return printAuthStatus(authStatusResponse{
				OK:           true,
				LoggedIn:     false,
				Message:      "Run `distlang helpers login` first.",
				AuthBaseURL:  auth.ResolveBaseURL(),
				StoreBaseURL: storeapi.ResolveBaseURL(),
			}, jsonOutput)
		}
		return printAuthStatus(authStatusResponse{
			OK:           false,
			LoggedIn:     false,
			Error:        "auth_status_failed",
			Message:      err.Error(),
			AuthBaseURL:  auth.ResolveBaseURL(),
			StoreBaseURL: storeapi.ResolveBaseURL(),
		}, jsonOutput)
	}

	whoami, err := client.WhoAmI(session.AccessToken)
	if err != nil {
		return printAuthStatus(authStatusResponse{
			OK:           false,
			LoggedIn:     false,
			Error:        "auth_status_failed",
			Message:      err.Error(),
			AuthBaseURL:  auth.ResolveBaseURL(),
			StoreBaseURL: storeapi.ResolveBaseURL(),
		}, jsonOutput)
	}

	return printAuthStatus(authStatusResponse{
		OK:       true,
		LoggedIn: true,
		User: &authStatusUser{
			ID:    whoami.User.ID,
			Email: whoami.User.Email,
			Name:  whoami.User.Name,
		},
		Token: &authStatusToken{
			ExpiresAt: whoami.Token.ExpiresAt,
			Scope:     whoami.Token.Scope,
		},
		AuthBaseURL:  auth.ResolveBaseURL(),
		StoreBaseURL: storeapi.ResolveBaseURL(),
	}, jsonOutput)
}

type authStatusResponse struct {
	OK           bool             `json:"ok"`
	LoggedIn     bool             `json:"logged_in"`
	Error        string           `json:"error,omitempty"`
	Message      string           `json:"message,omitempty"`
	User         *authStatusUser  `json:"user,omitempty"`
	Token        *authStatusToken `json:"token,omitempty"`
	AuthBaseURL  string           `json:"auth_base_url"`
	StoreBaseURL string           `json:"store_base_url"`
}

type authStatusUser struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

type authStatusToken struct {
	ExpiresAt string `json:"expires_at,omitempty"`
	Scope     string `json:"scope,omitempty"`
}

func printAuthStatus(response authStatusResponse, jsonOutput bool) int {
	if jsonOutput {
		payload, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "helpers auth status failed: %v\n", err)
			return 1
		}
		fmt.Println(string(payload))
		if response.OK {
			return 0
		}
		return 1
	}

	if !response.OK && response.Message != "" {
		fmt.Fprintf(os.Stderr, "helpers auth status failed: %s\n", response.Message)
		return 1
	}
	if !response.LoggedIn {
		fmt.Println("Logged in: no")
		if response.Message != "" {
			fmt.Println(response.Message)
		}
		fmt.Printf("Auth base URL: %s\n", response.AuthBaseURL)
		fmt.Printf("Store base URL: %s\n", response.StoreBaseURL)
		return 0
	}
	fmt.Println("Logged in: yes")
	if response.User != nil {
		if response.User.Name != "" {
			fmt.Printf("User: %s <%s>\n", response.User.Name, response.User.Email)
		} else {
			fmt.Printf("User: %s\n", response.User.Email)
		}
	}
	if response.Token != nil && response.Token.ExpiresAt != "" {
		fmt.Printf("Token expires at: %s\n", response.Token.ExpiresAt)
	}
	fmt.Printf("Auth base URL: %s\n", response.AuthBaseURL)
	fmt.Printf("Store base URL: %s\n", response.StoreBaseURL)
	return 0
}
