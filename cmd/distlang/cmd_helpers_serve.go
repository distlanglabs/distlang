package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/distlanglabs/distlang/pkg/helpers/mockserver"
)

func runHelpersServe(args []string) int {
	host := "127.0.0.1"
	port := 9191
	mode := "mock"

	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help":
			commandHelpHelpersServe()
			return 0
		case arg == "--mock":
			mode = "mock"
		case strings.HasPrefix(arg, "--host="):
			host = strings.TrimSpace(strings.TrimPrefix(arg, "--host="))
		case strings.HasPrefix(arg, "--port="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--port="))
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed <= 0 || parsed > 65535 {
				fmt.Fprintf(os.Stderr, "invalid port: %s\n", value)
				return 1
			}
			port = parsed
		default:
			fmt.Fprintf(os.Stderr, "unknown helpers serve flag: %s\n", arg)
			return 1
		}
	}

	if mode != "mock" {
		fmt.Fprintf(os.Stderr, "unsupported helpers serve mode: %s\n", mode)
		return 1
	}

	running, err := mockserver.Start(mockserver.Config{Host: host, Port: port})
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers serve failed: %v\n", err)
		return 1
	}
	defer func() {
		_ = running.Close(context.Background())
	}()

	fmt.Printf("helpers mock server listening on %s\n", running.URL())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	if err := running.Close(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "helpers serve shutdown failed: %v\n", err)
		return 1
	}
	return 0
}
