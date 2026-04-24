package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/distlanglabs/distlang/pkg/auth"
	storeapi "github.com/distlanglabs/distlang/pkg/store"
)

func runHelpersRequest(args []string) int {
	if len(args) == 0 {
		commandHelpHelpersRequest()
		return 1
	}

	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		commandHelpHelpersRequest()
		return 0
	}

	method := ""
	pathOrURL := ""
	bodyFile := ""
	contentType := ""
	baseURL := storeapi.ResolveBaseURL()
	jsonOutput := false

	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help":
			commandHelpHelpersRequest()
			return 0
		case arg == "--json":
			jsonOutput = true
		case strings.HasPrefix(arg, "--body-file="):
			bodyFile = strings.TrimPrefix(arg, "--body-file=")
		case strings.HasPrefix(arg, "--content-type="):
			contentType = strings.TrimPrefix(arg, "--content-type=")
		case strings.HasPrefix(arg, "--base-url="):
			baseURL = strings.TrimRight(strings.TrimSpace(strings.TrimPrefix(arg, "--base-url=")), "/")
		case strings.HasPrefix(arg, "-"):
			return printRequestError(jsonOutput, "invalid_flag", fmt.Sprintf("unknown helpers request flag: %s", arg))
		case method == "":
			method = strings.ToUpper(strings.TrimSpace(arg))
		case pathOrURL == "":
			pathOrURL = strings.TrimSpace(arg)
		default:
			return printRequestError(jsonOutput, "invalid_arguments", "helpers request accepts only METHOD PATH_OR_URL and optional flags")
		}
	}

	if method == "" || pathOrURL == "" {
		return printRequestError(jsonOutput, "invalid_arguments", "usage: distlang helpers request <METHOD> <PATH_OR_URL> [--body-file=path] [--content-type=type] [--base-url=url] [--json]")
	}

	requestURL, err := resolveRequestURL(baseURL, pathOrURL)
	if err != nil {
		return printRequestError(jsonOutput, "invalid_request_url", err.Error())
	}

	var body []byte
	if bodyFile != "" {
		body, err = os.ReadFile(bodyFile)
		if err != nil {
			return printRequestError(jsonOutput, "body_file_read_failed", err.Error())
		}
	}

	authClient := auth.NewClient(auth.ResolveBaseURL())
	session, err := authClient.EnsureSession()
	if err != nil {
		if errors.Is(err, auth.ErrNotLoggedIn) {
			return printRequestError(jsonOutput, "not_logged_in", "Run `distlang helpers login` first.")
		}
		return printRequestError(jsonOutput, "auth_session_failed", err.Error())
	}

	req, err := http.NewRequest(method, requestURL, bytes.NewReader(body))
	if err != nil {
		return printRequestError(jsonOutput, "request_build_failed", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return printRequestError(jsonOutput, "request_failed", err.Error())
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, 10<<20))
	if err != nil {
		return printRequestError(jsonOutput, "response_read_failed", err.Error())
	}

	if jsonOutput {
		return printRequestJSONEnvelope(res, responseBody)
	}

	if _, err := os.Stdout.Write(responseBody); err != nil {
		fmt.Fprintf(os.Stderr, "helpers request failed: %v\n", err)
		return 1
	}
	if len(responseBody) == 0 || responseBody[len(responseBody)-1] != '\n' {
		fmt.Println()
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return 0
	}
	return 1
}

type helpersRequestResponse struct {
	OK      bool              `json:"ok"`
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
	Error   string            `json:"error,omitempty"`
	Message string            `json:"message,omitempty"`
}

func printRequestJSONEnvelope(res *http.Response, responseBody []byte) int {
	envelope := helpersRequestResponse{
		OK:      res.StatusCode >= 200 && res.StatusCode < 300,
		Status:  res.StatusCode,
		Headers: flattenHeaders(res.Header),
		Body:    parseResponseBody(responseBody),
	}
	payload, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "helpers request failed: %v\n", err)
		return 1
	}
	fmt.Println(string(payload))
	if envelope.OK {
		return 0
	}
	return 1
}

func printRequestError(jsonOutput bool, code, message string) int {
	if jsonOutput {
		payload, err := json.MarshalIndent(helpersRequestResponse{
			OK:      false,
			Error:   code,
			Message: message,
		}, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "helpers request failed: %v\n", err)
			return 1
		}
		fmt.Println(string(payload))
		return 1
	}
	fmt.Fprintf(os.Stderr, "helpers request failed: %s\n", message)
	return 1
}

func resolveRequestURL(baseURL, pathOrURL string) (string, error) {
	if strings.HasPrefix(pathOrURL, "https://") || strings.HasPrefix(pathOrURL, "http://") {
		parsed, err := url.Parse(pathOrURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return "", fmt.Errorf("invalid request URL: %s", pathOrURL)
		}
		return parsed.String(), nil
	}
	if !strings.HasPrefix(pathOrURL, "/") {
		return "", fmt.Errorf("path must start with / or be a full URL")
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid base URL: %s", baseURL)
	}
	rel, err := url.Parse(pathOrURL)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", pathOrURL)
	}
	return base.ResolveReference(rel).String(), nil
}

func flattenHeaders(headers http.Header) map[string]string {
	out := map[string]string{}
	for key, values := range headers {
		out[strings.ToLower(key)] = strings.Join(values, ", ")
	}
	return out
}

func parseResponseBody(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	var parsed any
	if json.Unmarshal(body, &parsed) == nil {
		return parsed
	}
	return string(body)
}
