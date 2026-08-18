package workerd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner launches a local workerd preview process.
type Runner struct{}

func New() Runner {
	return Runner{}
}

// Start launches workerd against the provided worker entrypoint.
func (Runner) Start(ctx context.Context, entryPath string, port int) error {
	if _, err := exec.LookPath("workerd"); err != nil {
		return fmt.Errorf("workerd not found in PATH")
	}

	tmpDir, err := os.MkdirTemp("", "distlang-workerd-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	workerCopy := filepath.Join(tmpDir, "worker.js")
	data, err := os.ReadFile(entryPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(workerCopy, data, 0o644); err != nil {
		return err
	}

	config := filepath.Join(tmpDir, "workerd.capnp")
	content := fmt.Sprintf(`using Workerd = import "/workerd/workerd.capnp";
const config :Workerd.Config = (
  services = [
    (
      name = "internet",
      network = (
        allow = [ "public", "private" ],
        tlsOptions = (trustBrowserCas = true)
      )
    ),
    (
      name = "distlang",
      worker = (
        compatibilityDate = "2024-01-01",
        modules = [ (name = "worker", esModule = embed "worker.js") ]%s
      )
    )
  ],
  sockets = [
    ( name = "http", address = "127.0.0.1:%d", http = (), service = "distlang" )
  ]
);
`, workerdBindings(), port)
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "workerd", "serve", config, "config")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func workerdBindings() string {
	keys := []string{"DISTLANG_HELPERS_MODE", "DISTLANG_STORE_BASE_URL", "DISTLANG_SERVICE_TOKEN"}
	bindings := []string{}
	for _, key := range keys {
		value := os.Getenv(key)
		if value == "" {
			continue
		}
		bindings = append(bindings, fmt.Sprintf(`(name = %q, text = %q)`, key, value))
	}
	if len(bindings) == 0 {
		return ""
	}
	return ",\n        bindings = [ " + strings.Join(bindings, ", ") + " ]"
}
