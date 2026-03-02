package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/distlanglabs/distlang/pkg/passes"
	"github.com/distlanglabs/distlang/pkg/runtime"
	runtimetypes "github.com/distlanglabs/distlang/pkg/runtime/types"
)

func runRun(args []string) int {
	if len(args) >= 1 && (args[0] == "--help" || args[0] == "-h") {
		commandHelpRun()
		return 0
	}

	port := 5656
	filePath := ""

	for _, arg := range args {
		if strings.HasPrefix(arg, "--port=") {
			val := strings.TrimPrefix(arg, "--port=")
			p, err := strconv.Atoi(val)
			if err != nil || p <= 0 || p > 65535 {
				fmt.Fprintf(os.Stderr, "invalid port: %s\n", val)
				return 1
			}
			port = p
			continue
		}

		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			return 1
		}

		if filePath == "" {
			filePath = arg
		} else {
			fmt.Fprintln(os.Stderr, "run accepts only one file path")
			return 1
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "run requires a file path")
		return 1
	}

	result, err := passes.Execute(filePath, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	engine := runtime.NewDefaultEngine()
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		headers := map[string]string{}
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		resp, err := engine.RunWorker(filePath, result.Emitted, runtimetypes.Request{
			URL:     r.URL.String(),
			Method:  r.Method,
			Headers: headers,
			Body:    string(body),
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("worker error: %v", err), http.StatusInternalServerError)
			return
		}

		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
		if resp.Status == 0 {
			resp.Status = http.StatusOK
		}
		w.WriteHeader(resp.Status)
		_, _ = w.Write([]byte(resp.Body))
	})

	fmt.Printf("Serving worker %s at http://%s\n", filePath, addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		return 1
	}

	return 0
}
