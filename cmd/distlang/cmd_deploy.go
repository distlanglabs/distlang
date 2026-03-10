package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/backend"
	cloudflareprovider "github.com/distlanglabs/distlang/pkg/provider/cloudflare"
)

func runDeploy(args []string) int {
	if len(args) >= 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpDeploy()
		return 0
	}

	target := "cloudflare"
	filePath := ""

	for _, arg := range args {
		if strings.HasPrefix(arg, "--target=") {
			target = strings.TrimSpace(strings.TrimPrefix(arg, "--target="))
			continue
		}

		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "unknown deploy flag: %s\n", arg)
			return 1
		}

		if filePath == "" {
			filePath = arg
		} else {
			fmt.Fprintln(os.Stderr, "deploy accepts only one file path")
			return 1
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "deploy requires a file path")
		fmt.Fprintln(os.Stderr, "Usage: distlang deploy <file> [--target=cloudflare]")
		return 1
	}

	if target != "cloudflare" {
		fmt.Fprintf(os.Stderr, "unsupported deploy target: %s\n", target)
		return 1
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: resolve file path: %v\n", err)
		return 1
	}
	projectDir := filepath.Dir(absFilePath)

	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: resolve working directory: %v\n", err)
		return 1
	}
	if err := os.Chdir(projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: enter project directory: %v\n", err)
		return 1
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	envPath := filepath.Join("targets", "cloudflare", "cloudflare.env")
	deployEnv, err := cloudflareDeployEnv(envPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: %v\n", err)
		return 1
	}

	v8Out, err := backend.BuildV8(absFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: build v8 backend: %v\n", err)
		return 1
	}

	res, err := cloudflareprovider.Package(v8Out, cloudflareprovider.Context{
		ProjectName:   fileBase(absFilePath),
		KVBindingName: "DISTLANG_KV",
		KVNamespaceID: strings.TrimSpace(deployEnv["CLOUDFLARE_KV_NAMESPACE_ID"]),
		KVPreviewID:   strings.TrimSpace(deployEnv["CLOUDFLARE_KV_PREVIEW_ID"]),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: package cloudflare artifacts: %v\n", err)
		return 1
	}

	if err := artifacts.WriteAll(res); err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: write artifacts: %v\n", err)
		return 1
	}

	if _, err := exec.LookPath("wrangler"); err != nil {
		fmt.Fprintln(os.Stderr, "deploy failed: wrangler not found in PATH")
		fmt.Fprintln(os.Stderr, "install it with: npm install -g wrangler")
		return 1
	}

	fmt.Println("Deploying to Cloudflare...")
	cmd := exec.Command("wrangler", "deploy")
	cmd.Dir = filepath.Join("dist", "cloudflare")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = mergedEnv(deployEnv)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: %v\n", err)
		return 1
	}

	return 0
}

func cloudflareDeployEnv(envPath string) (map[string]string, error) {
	required := []string{"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID"}
	optional := []string{"CLOUDFLARE_KV_NAMESPACE_ID", "CLOUDFLARE_KV_PREVIEW_ID"}

	fromFile, err := loadEnvFile(envPath)
	if err != nil {
		return nil, err
	}

	values := map[string]string{}
	for _, key := range required {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			values[key] = v
			continue
		}
		if v := strings.TrimSpace(fromFile[key]); v != "" {
			values[key] = v
		}
	}
	for _, key := range optional {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			values[key] = v
			continue
		}
		if v := strings.TrimSpace(fromFile[key]); v != "" {
			values[key] = v
		}
	}

	missing := missingKeys(required, values)
	if len(missing) == 0 {
		return values, nil
	}

	fmt.Fprintf(os.Stderr, "warning: missing required Cloudflare deploy env vars: %s\n", strings.Join(missing, ", "))
	if isInteractive() {
		if err := promptMissingValues(missing, values); err != nil {
			return nil, err
		}
		missing = missingKeys(required, values)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s (set them in shell env or %s)", strings.Join(missing, ", "), envPath)
	}

	return values, nil
}
func loadEnvFile(path string) (map[string]string, error) {
	values := map[string]string{}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return values, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNo)
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("%s:%d: empty key", path, lineNo)
		}

		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")
		value = strings.Trim(value, "'")
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

func missingKeys(required []string, values map[string]string) []string {
	var missing []string
	for _, key := range required {
		if strings.TrimSpace(values[key]) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func promptMissingValues(keys []string, values map[string]string) error {
	sort.Strings(keys)
	reader := bufio.NewReader(os.Stdin)
	for _, key := range keys {
		fmt.Fprintf(os.Stderr, "Enter %s: ", key)
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		values[key] = line
	}
	return nil
}

func mergedEnv(extra map[string]string) []string {
	env := map[string]string{}
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		env[parts[0]] = parts[1]
	}

	for k, v := range extra {
		env[k] = v
	}

	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
