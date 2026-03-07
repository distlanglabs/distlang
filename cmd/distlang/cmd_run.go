package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/backend"
	wasmrt "github.com/distlanglabs/distlang/pkg/runtime/wasmtime"
	v8rt "github.com/distlanglabs/distlang/pkg/runtime/workerd"
)

func runRun(args []string) int {
	if len(args) >= 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpRun()
		return 0
	}

	v8Port := 5656
	wasmPort := 5757
	filePath := ""

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--v8-port="):
			val := strings.TrimPrefix(arg, "--v8-port=")
			port, err := parsePort(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid v8 port: %s\n", val)
				return 1
			}
			v8Port = port
		case strings.HasPrefix(arg, "--wasm-port="):
			val := strings.TrimPrefix(arg, "--wasm-port=")
			port, err := parsePort(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid wasm port: %s\n", val)
				return 1
			}
			wasmPort = port
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			return 1
		case filePath == "":
			filePath = arg
		default:
			fmt.Fprintln(os.Stderr, "run accepts only one file path")
			return 1
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "run requires a file path")
		return 1
	}

	v8Out, err := backend.BuildV8(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: build v8 backend: %v\n", err)
		return 1
	}
	if err := artifacts.WriteAll(v8Out.Artifacts); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: write v8 artifacts: %v\n", err)
		return 1
	}

	wasmOut, err := backend.BuildWasm(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: build wasm backend: %v\n", err)
		return 1
	}
	if err := artifacts.WriteAll(wasmOut.Artifacts); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: write wasm artifacts: %v\n", err)
		return 1
	}

	if _, err := exec.LookPath("workerd"); err != nil {
		fmt.Fprintln(os.Stderr, "run failed: workerd not found in PATH")
		return 1
	}
	if _, err := exec.LookPath("wasmtime"); err != nil {
		fmt.Fprintln(os.Stderr, "run failed: wasmtime not found in PATH")
		return 1
	}
	if filepath.Ext(wasmOut.EntryPath) != ".wasm" {
		fmt.Fprintf(os.Stderr, "run failed: wasm backend is not runnable yet; expected a .wasm artifact, got %s\n", wasmOut.EntryPath)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error, 2)
	v8Runner := v8rt.New()
	wasmRunner := wasmrt.New()

	go func() {
		errs <- fmt.Errorf("v8 runtime: %w", v8Runner.Start(ctx, v8Out.EntryPath, v8Port))
	}()
	go func() {
		errs <- fmt.Errorf("wasm runtime: %w", wasmRunner.Start(ctx, wasmOut.EntryPath, wasmPort))
	}()

	fmt.Printf("Starting local runtimes for %s\n", filePath)
	fmt.Printf("- v8:   http://127.0.0.1:%d\n", v8Port)
	fmt.Printf("- wasm: http://127.0.0.1:%d\n", wasmPort)

	err = <-errs
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	return 0
}

func parsePort(val string) (int, error) {
	port, err := strconv.Atoi(val)
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port")
	}
	return port, nil
}
