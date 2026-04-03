package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/distlanglabs/distlang/pkg/artifacts"
	"github.com/distlanglabs/distlang/pkg/auth"
	"github.com/distlanglabs/distlang/pkg/backend"
	"github.com/distlanglabs/distlang/pkg/deployclient"
	cloudflareprovider "github.com/distlanglabs/distlang/pkg/provider/cloudflare"
	"github.com/distlanglabs/distlang/pkg/store"
)

func runDeploy(args []string) int {
	if len(args) >= 1 && args[0] == "debug" {
		if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
			commandHelpDeployDebug()
			return 0
		}
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "Usage: distlang deploy debug <file>")
			return 1
		}
		return runDeployDebug(args[1])
	}

	if len(args) >= 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpDeploy()
		return 0
	}

	target := "distlang"
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
		fmt.Fprintln(os.Stderr, "Usage: distlang deploy <file> [--target=distlang|cloudflare]")
		return 1
	}

	if target == "distlang" {
		return runHostedDeploy(filePath)
	}
	if target != "cloudflare" {
		fmt.Fprintf(os.Stderr, "unsupported deploy target: %s\n", target)
		return 1
	}

	return runCloudflareDeploy(filePath)
}

func runHostedDeploy(filePath string) int {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: resolve file path: %v\n", err)
		return 1
	}
	v8Out, err := backend.BuildV8(absFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: build v8 backend: %v\n", err)
		return 1
	}
	if len(v8Out.Workers) > 0 {
		fmt.Fprintln(os.Stderr, "deploy failed: hosted Distlang deploy currently supports single-worker apps only")
		return 1
	}
	authClient := auth.NewClient(auth.ResolveBaseURL())
	session, err := authClient.EnsureSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: hosted deploy requires login: %v\n", err)
		return 1
	}
	serviceToken, err := resolveHelpersServiceToken(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: %v\n", err)
		return 1
	}
	client := deployclient.New(store.ResolveBaseURL())
	request := deployclient.CreateDeploymentRequest{
		App:            inferredAppName(absFilePath),
		MetricsBuckets: inferMetricsBuckets(v8Out.Emitted),
		Provider:       "cloudflare",
		ServiceToken:   serviceToken,
		CLIVersion:     version,
		CLICommit:      commit,
	}
	request.Worker.Kind = "single"
	request.Worker.Code = v8Out.Emitted
	response, err := client.CreateDeployment(session.AccessToken, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: %v\n", err)
		return 1
	}
	fmt.Printf("Hosted deploy succeeded\n- app: %s\n- script: %s\n- hostname: %s\n- url: %s\n", response.Deployment.App, response.Deployment.ScriptName, response.Deployment.Hostname, response.Deployment.URL)
	return 0
}

func runDeployDebug(filePath string) int {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy debug failed: resolve file path: %v\n", err)
		return 1
	}

	appName := inferredAppName(absFilePath)
	fmt.Println("Deploy debug")
	fmt.Printf("- file: %s\n", absFilePath)
	fmt.Printf("- app: %s\n", appName)

	fmt.Println("\nBuild")
	v8Out, err := backend.BuildV8(absFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy debug failed: build v8 backend: %v\n", err)
		return 1
	}
	workerMode := "single"
	if len(v8Out.Workers) > 0 {
		workerMode = fmt.Sprintf("multi (%d workers)", len(v8Out.Workers))
	}
	fmt.Printf("- backend: v8\n")
	fmt.Printf("- workers: %s\n", workerMode)
	fmt.Printf("- emitted bytes: %d\n", len(v8Out.Emitted))
	if len(v8Out.Workers) > 0 {
		fmt.Println("\nResult")
		fmt.Println("- hosted deploy is not ready")
		fmt.Println("- reason: hosted Distlang deploy currently supports single-worker apps only")
		return 1
	}

	authClient := auth.NewClient(auth.ResolveBaseURL())
	session, err := authClient.EnsureSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy debug failed: hosted deploy requires login: %v\n", err)
		return 1
	}
	whoami, err := authClient.WhoAmI(session.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy debug failed: auth whoami: %v\n", err)
		return 1
	}

	fmt.Println("\nSession")
	fmt.Printf("- auth base: %s\n", auth.ResolveBaseURL())
	fmt.Printf("- user: %s <%s>\n", whoami.User.Name, whoami.User.Email)
	fmt.Printf("- user id: %s\n", whoami.User.ID)
	fmt.Printf("- expires: %s\n", session.ExpiresAt.UTC().Format(time.RFC3339))

	serviceTokenResponse, err := authClient.ServiceToken(session.AccessToken, "objectdb", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy debug failed: fetch service token: %v\n", err)
		return 1
	}
	serviceToken := strings.TrimSpace(serviceTokenResponse.AccessToken)
	fmt.Println("\nService token")
	fmt.Printf("- mint: ok\n")
	fmt.Printf("- kind: %s\n", describeTokenKind(serviceToken))
	fmt.Printf("- token: %s\n", redactToken(serviceToken))
	fmt.Printf("- length: %d\n", len(serviceToken))

	tokenWhoAmI, err := authClient.ServiceTokenWhoAmI(serviceToken)
	if err != nil {
		fmt.Println("\nService token whoami")
		fmt.Printf("- status: failed\n")
		fmt.Printf("- error: %v\n", err)
		fmt.Println("\nResult")
		fmt.Println("- hosted deploy is not ready")
		return 1
	}
	fmt.Println("\nService token whoami")
	fmt.Printf("- status: ok\n")
	fmt.Printf("- user: %s <%s>\n", tokenWhoAmI.User.Name, tokenWhoAmI.User.Email)
	fmt.Printf("- service: %s\n", tokenWhoAmI.Token.Service)
	fmt.Printf("- scope: %s\n", tokenWhoAmI.Token.Scope)

	storeClient := store.NewClient(store.ResolveBaseURL())
	status, err := storeClient.ObjectDBStatus(serviceToken)
	if err != nil {
		fmt.Println("\nStore auth with service token")
		fmt.Printf("- base: %s\n", storeClient.BaseURL())
		fmt.Printf("- status: failed\n")
		fmt.Printf("- error: %v\n", err)
		fmt.Println("\nResult")
		fmt.Println("- hosted deploy is not ready")
		return 1
	}

	fmt.Println("\nStore auth with service token")
	fmt.Printf("- base: %s\n", storeClient.BaseURL())
	fmt.Printf("- status: ok\n")
	fmt.Printf("- service: %s %s\n", status.Service, status.Version)
	fmt.Printf("- user: %s <%s>\n", status.User.Name, status.User.Email)

	listReq, err := newDeploymentsListRequest(session.AccessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy debug failed: %v\n", err)
		return 1
	}
	listRes, err := deployClientDebugDo(listReq)
	if err != nil {
		fmt.Println("\nDeployments API")
		fmt.Printf("- status: failed\n")
		fmt.Printf("- error: %v\n", err)
		fmt.Println("\nResult")
		fmt.Println("- hosted deploy is not ready")
		return 1
	}
	fmt.Println("\nDeployments API")
	fmt.Printf("- status: ok\n")
	fmt.Printf("- deployments visible: %d\n", listRes)

	fmt.Println("\nResult")
	fmt.Println("- hosted deploy debug checks passed")
	return 0
}

func runCloudflareDeploy(filePath string) int {
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

	usesHelpers := sourceUsesDistlangHelpers(absFilePath)
	serviceToken := ""
	if usesHelpers {
		serviceToken, err = resolveHelpersServiceToken(deployEnv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "deploy failed: %v\n", err)
			return 1
		}
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
		StoreBaseURL:  strings.TrimSpace(deployEnv["DISTLANG_STORE_BASE_URL"]),
		HelpersMode:   helpersModeForDeploy(deployEnv, usesHelpers),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: package cloudflare artifacts: %v\n", err)
		return 1
	}

	if err := artifacts.WriteAll(res); err != nil {
		fmt.Fprintf(os.Stderr, "deploy failed: write artifacts: %v\n", err)
		return 1
	}

	deployDirs := []string{filepath.Join("dist", "cloudflare")}
	if len(v8Out.Workers) > 0 {
		deployDirs = deployDirs[:0]
		for _, worker := range v8Out.Workers {
			deployDirs = append(deployDirs, filepath.Join("dist", "cloudflare", worker.Name))
		}
	}

	if _, err := exec.LookPath("wrangler"); err != nil {
		fmt.Fprintln(os.Stderr, "deploy failed: wrangler not found in PATH")
		fmt.Fprintln(os.Stderr, "install it with: npm install -g wrangler")
		return 1
	}

	if usesHelpers {
		for _, deployDir := range deployDirs {
			if err := putWranglerSecret(deployDir, deployEnv, "DISTLANG_SERVICE_TOKEN", serviceToken); err != nil {
				fmt.Fprintf(os.Stderr, "deploy failed: %v\n", err)
				return 1
			}
		}
	}

	for _, deployDir := range deployDirs {
		fmt.Printf("Deploying to Cloudflare (%s)...\n", deployDir)
		cmd := exec.Command("wrangler", "deploy")
		cmd.Dir = deployDir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = mergedEnv(deployEnv)

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "deploy failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func inferredAppName(filePath string) string {
	projectDir := filepath.Dir(filePath)
	name := filepath.Base(projectDir)
	if strings.TrimSpace(name) == "" || name == "." || name == string(filepath.Separator) {
		return fileBase(filePath)
	}
	return name
}

func sourceUsesDistlangHelpers(filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	source := string(data)
	return strings.Contains(source, `from "distlang"`) || strings.Contains(source, "from 'distlang'")
}

func resolveHelpersServiceToken(deployEnv map[string]string) (string, error) {
	if token := strings.TrimSpace(os.Getenv("DISTLANG_SERVICE_TOKEN")); token != "" {
		return token, nil
	}
	if token := strings.TrimSpace(deployEnv["DISTLANG_SERVICE_TOKEN"]); token != "" {
		return token, nil
	}

	authClient := auth.NewClient(auth.ResolveBaseURL())
	session, err := authClient.EnsureSession()
	if err != nil {
		return "", fmt.Errorf("helpers.ObjectDB requires login for deploy: %w", err)
	}
	serviceToken, err := authClient.ServiceToken(session.AccessToken, "objectdb", false)
	if err != nil {
		return "", fmt.Errorf("fetch service token: %w", err)
	}
	return strings.TrimSpace(serviceToken.AccessToken), nil
}

func describeTokenKind(token string) string {
	if strings.HasPrefix(token, "dsts_") {
		return "opaque service token"
	}
	if strings.Count(token, ".") == 2 {
		return "jwt"
	}
	return "unknown"
}

func redactToken(token string) string {
	if len(token) <= 18 {
		return token
	}
	return token[:12] + "..." + token[len(token)-6:]
}

func inferMetricsBuckets(emitted string) []string {
	if strings.TrimSpace(emitted) == "" {
		return nil
	}
	set := map[string]struct{}{}
	for _, bucket := range findInstantiateMetricsBuckets(emitted) {
		if bucket == "" {
			continue
		}
		set[bucket] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	buckets := make([]string, 0, len(set))
	for bucket := range set {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	return buckets
}

func findInstantiateMetricsBuckets(emitted string) []string {
	const marker = "instantiateMetrics("
	buckets := []string{}
	for start := 0; start < len(emitted); {
		idx := strings.Index(emitted[start:], marker)
		if idx < 0 {
			break
		}
		open := start + idx + len(marker) - 1
		close := matchingParenIndex(emitted, open)
		if close < 0 {
			break
		}
		args := splitTopLevelArgs(emitted[open+1 : close])
		if len(args) >= 2 {
			if bucket, ok := trimQuotedString(args[1]); ok {
				buckets = append(buckets, bucket)
			}
		}
		start = close + 1
	}
	return buckets
}

func matchingParenIndex(s string, open int) int {
	depth := 0
	quote := byte(0)
	escaped := false
	for i := open; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTopLevelArgs(s string) []string {
	parts := []string{}
	start := 0
	parenDepth := 0
	braceDepth := 0
	bracketDepth := 0
	quote := byte(0)
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
			continue
		}
		switch ch {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case ',':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}

func trimQuotedString(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) < 2 {
		return "", false
	}
	quote := trimmed[0]
	if (quote != '\'' && quote != '"') || trimmed[len(trimmed)-1] != quote {
		return "", false
	}
	return trimmed[1 : len(trimmed)-1], true
}

func newDeploymentsListRequest(accessToken string) (*http.Request, error) {
	req, err := http.NewRequest("GET", strings.TrimRight(store.ResolveBaseURL(), "/")+"/deployments/v1", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return req, nil
}

func deployClientDebugDo(req *http.Request) (int, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		return 0, fmt.Errorf("deployments request failed (%s): %s", res.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Deployments []json.RawMessage `json:"deployments"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return len(payload.Deployments), nil
}

func putWranglerSecret(dir string, env map[string]string, key, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("missing value for secret %s", key)
	}
	fmt.Printf("Updating Cloudflare secret %s...\n", key)
	cmd := exec.Command("wrangler", "secret", "put", key)
	cmd.Dir = dir
	cmd.Env = mergedEnv(env)
	cmd.Stdin = strings.NewReader(value + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("set wrangler secret %s: %w", key, err)
	}
	return nil
}

func helpersModeForDeploy(deployEnv map[string]string, usesHelpers bool) string {
	if mode := strings.TrimSpace(os.Getenv("DISTLANG_HELPERS_MODE")); mode != "" {
		return mode
	}
	if mode := strings.TrimSpace(deployEnv["DISTLANG_HELPERS_MODE"]); mode != "" {
		return mode
	}
	if usesHelpers {
		return "live"
	}
	return ""
}

func cloudflareDeployEnv(envPath string) (map[string]string, error) {
	required := []string{"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID"}
	optional := []string{"CLOUDFLARE_KV_NAMESPACE_ID", "CLOUDFLARE_KV_PREVIEW_ID", "DISTLANG_STORE_BASE_URL", "DISTLANG_HELPERS_MODE", "DISTLANG_SERVICE_TOKEN"}

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
