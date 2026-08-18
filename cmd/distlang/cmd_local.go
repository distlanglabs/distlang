package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	localserver "github.com/distlanglabs/distlang/pkg/local/server"
)

func runLocal(args []string) int {
	host := "127.0.0.1"
	port := 4817
	openBrowser := true

	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help":
			commandHelpLocal()
			return 0
		case arg == "--no-open":
			openBrowser = false
		case strings.HasPrefix(arg, "--host="):
			host = strings.TrimSpace(strings.TrimPrefix(arg, "--host="))
		case strings.HasPrefix(arg, "--port="):
			parsed, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "--port=")))
			if err != nil || parsed <= 0 || parsed > 65535 {
				fmt.Fprintf(os.Stderr, "invalid port: %s\n", strings.TrimPrefix(arg, "--port="))
				return 1
			}
			port = parsed
		default:
			fmt.Fprintf(os.Stderr, "unknown local flag: %s\n", arg)
			return 1
		}
	}

	running, err := localserver.Start(localserver.Config{Host: host, Port: port})
	if err != nil {
		fmt.Fprintf(os.Stderr, "local failed: %v\n", err)
		return 1
	}
	defer func() { _ = running.Close(context.Background()) }()

	fmt.Printf("distlang local listening on %s\n", running.URL())
	fmt.Printf("- local UI: %s\n", running.DistlangURL())
	fmt.Printf("- metrics API: %s/metrics/v1\n", running.DistlangURL())
	fmt.Println("- storage: memory")
	if openBrowser {
		fmt.Println("- open: not implemented yet; visit the local UI URL manually")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	if err := running.Close(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "local shutdown failed: %v\n", err)
		return 1
	}
	return 0
}
