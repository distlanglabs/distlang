package wasmtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner launches a local wasmtime preview process.
type Runner struct{}

func New() Runner {
	return Runner{}
}

// Start launches wasmtime serve when a runnable wasm module is present.
func (Runner) Start(ctx context.Context, entryPath string, port int) error {
	if _, err := exec.LookPath("wasmtime"); err != nil {
		return fmt.Errorf("wasmtime not found in PATH")
	}

	if filepath.Ext(entryPath) != ".wasm" {
		return fmt.Errorf("wasmtime runtime requires a .wasm entry artifact, got %s", entryPath)
	}

	cmd := exec.CommandContext(ctx, "wasmtime", "serve", "--addr", fmt.Sprintf("127.0.0.1:%d", port), entryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
