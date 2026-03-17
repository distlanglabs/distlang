package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/backend"
	v8rt "github.com/distlanglabs/distlang/pkg/runtime/workerd"
)

func runRun(args []string) int {
	if len(args) >= 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpRun()
		return 0
	}

	v8Port := 5656
	handlerSet1Port := 5656
	handlerSet2Port := 5657
	targetSet := "all"
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
		case strings.HasPrefix(arg, "--port1="):
			val := strings.TrimPrefix(arg, "--port1=")
			port, err := parsePort(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid handlerSet1 port: %s\n", val)
				return 1
			}
			handlerSet1Port = port
		case strings.HasPrefix(arg, "--port2="):
			val := strings.TrimPrefix(arg, "--port2=")
			port, err := parsePort(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid handlerSet2 port: %s\n", val)
				return 1
			}
			handlerSet2Port = port
		case strings.HasPrefix(arg, "--set="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--set="))
			if value != "all" && value != "handlerSet1" && value != "handlerSet2" {
				fmt.Fprintf(os.Stderr, "invalid set: %s (expected all, handlerSet1, or handlerSet2)\n", value)
				return 1
			}
			targetSet = value
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

	if _, err := exec.LookPath("workerd"); err != nil {
		fmt.Fprintln(os.Stderr, "run failed: workerd not found in PATH")
		return 1
	}

	if len(v8Out.Workers) > 0 {
		selected := []struct {
			name string
			path string
			port int
		}{}

		for _, worker := range v8Out.Workers {
			if targetSet != "all" && targetSet != worker.Name {
				continue
			}

			port := handlerSet1Port
			if worker.Name == "handlerSet2" {
				port = handlerSet2Port
			}

			selected = append(selected, struct {
				name string
				path string
				port int
			}{name: worker.Name, path: worker.EntryPath, port: port})
		}

		if len(selected) == 0 {
			fmt.Fprintf(os.Stderr, "run failed: no workers selected for --set=%s\n", targetSet)
			return 1
		}

		return runWorkers(filePath, selected)
	}

	if targetSet != "all" {
		fmt.Fprintln(os.Stderr, "run failed: --set only applies to simpleApp dual-worker programs")
		return 1
	}

	single := []struct {
		name string
		path string
		port int
	}{
		{name: "v8", path: v8Out.EntryPath, port: v8Port},
	}
	return runWorkers(filePath, single)
}

func runWorkers(filePath string, workers []struct {
	name string
	path string
	port int
}) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, len(workers))
	v8Runner := v8rt.New()

	for _, worker := range workers {
		w := worker
		go func() {
			errCh <- fmt.Errorf("%s runtime: %w", w.name, v8Runner.Start(ctx, w.path, w.port))
		}()
	}

	fmt.Printf("Starting local runtime for %s\n", filePath)
	for _, worker := range workers {
		fmt.Printf("- %s: http://127.0.0.1:%d\n", worker.name, worker.port)
	}

	err := <-errCh
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
