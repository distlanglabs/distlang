package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	localmetrics "github.com/distlanglabs/distlang/pkg/local/metrics"
	localserver "github.com/distlanglabs/distlang/pkg/local/server"
)

func runLocal(args []string) int {
	host := "127.0.0.1"
	port := 4817
	openBrowser := true
	useMemory := false
	dbPath := ""

	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help":
			commandHelpLocal()
			return 0
		case arg == "--no-open":
			openBrowser = false
		case arg == "--memory":
			useMemory = true
		case strings.HasPrefix(arg, "--db="):
			dbPath = strings.TrimSpace(strings.TrimPrefix(arg, "--db="))
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

	storage := "memory"
	if !useMemory {
		store, err := localmetrics.OpenSQLiteStore(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "local failed: open sqlite store: %v\n", err)
			return 1
		}
		storage = "sqlite"
		if dbPath == "" {
			if resolved, err := localmetrics.DefaultSQLitePath(); err == nil {
				dbPath = resolved
			}
		}
		running, err := localserver.Start(localserver.Config{Host: host, Port: port, Store: store, Backend: localmetrics.NewSQLiteQueryBackend(store)})
		if err != nil {
			fmt.Fprintf(os.Stderr, "local failed: %v\n", err)
			return 1
		}
		return waitForLocal(running, storage, dbPath, openBrowser)
	}

	running, err := localserver.Start(localserver.Config{Host: host, Port: port})
	if err != nil {
		fmt.Fprintf(os.Stderr, "local failed: %v\n", err)
		return 1
	}
	return waitForLocal(running, storage, dbPath, openBrowser)
}

func waitForLocal(running *localserver.Running, storage string, dbPath string, openBrowser bool) int {
	defer func() { _ = running.Close(context.Background()) }()
	fmt.Printf("distlang local listening on %s\n", running.URL())
	fmt.Printf("- local UI: %s\n", running.DistlangURL())
	fmt.Printf("- metrics API: %s/metrics/v1\n", running.DistlangURL())
	fmt.Printf("- storage: %s\n", storage)
	if dbPath != "" {
		fmt.Printf("- database: %s\n", dbPath)
	}
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
