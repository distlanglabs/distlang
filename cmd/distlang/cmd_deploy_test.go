package main

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestInferMetricsBuckets(t *testing.T) {
	tests := []struct {
		name    string
		emitted string
		want    []string
	}{
		{
			name:    "finds unique sorted buckets",
			emitted: `const a = instantiateMetrics({ reqs: "counter" }, "z-bucket"); const b = instantiateMetrics({ reqs: "counter" }, "a-bucket"); const c = instantiateMetrics({ reqs: "counter" }, "z-bucket");`,
			want:    []string{"a-bucket", "z-bucket"},
		},
		{
			name:    "ignores dynamic buckets",
			emitted: `const a = instantiateMetrics({ reqs: "counter" }, bucketName);`,
			want:    nil,
		},
		{
			name:    "supports single quotes",
			emitted: `const a = instantiateMetrics({ reqs: "counter" }, 'echo-metrics');`,
			want:    []string{"echo-metrics"},
		},
		{
			name: "supports multiline instantiate metrics call",
			emitted: `const appMetrics = helpers.instantiateMetrics(
				{
					echoReqCount: {
						kind: "counter",
						labels: ["route", "method", "status"],
					},
				},
				"app-echo-metrics",
			);`,
			want: []string{"app-echo-metrics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferMetricsBuckets(tt.emitted)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("inferMetricsBuckets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunDeployDeleteRequiresDeploymentID(t *testing.T) {
	stderr := captureStderr(t, func() {
		if code := runDeploy([]string{"delete"}); code != 1 {
			t.Fatalf("runDeploy(delete) = %d, want 1", code)
		}
	})
	if !strings.Contains(stderr, "Usage: distlang deploy delete <deployment-id>") {
		t.Fatalf("expected delete usage, got %q", stderr)
	}
}

func TestRunDeployListRejectsExtraArgs(t *testing.T) {
	stderr := captureStderr(t, func() {
		if code := runDeploy([]string{"list", "extra"}); code != 1 {
			t.Fatalf("runDeploy(list extra) = %d, want 1", code)
		}
	})
	if !strings.Contains(stderr, "Usage: distlang deploy list") {
		t.Fatalf("expected list usage, got %q", stderr)
	}
}

func TestRunDeployListHelp(t *testing.T) {
	stdout := captureStdout(t, func() {
		if code := runDeploy([]string{"list", "--help"}); code != 0 {
			t.Fatalf("runDeploy(list --help) = %d, want 0", code)
		}
	})
	if !strings.Contains(stdout, "deploy list - List hosted Distlang deployments") {
		t.Fatalf("expected list help, got %q", stdout)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	defer func() {
		os.Stdout = original
		_ = writer.Close()
	}()

	fn()
	_ = writer.Close()

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(output)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer reader.Close()

	os.Stderr = writer
	defer func() {
		os.Stderr = original
		_ = writer.Close()
	}()

	fn()
	_ = writer.Close()

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(output)
}
