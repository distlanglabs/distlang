package cloudflare

import (
	"testing"

	v8backend "github.com/distlanglabs/distlang/pkg/backend/v8"
)

func TestPackageProducesArtifacts(t *testing.T) {
	artifacts, err := Package(v8backend.Output{Emitted: "console.log('hi')"}, Context{
		ProjectName:   "example",
		KVBindingName: "DISTLANG_KV",
		KVNamespaceID: "namespace-123",
		KVPreviewID:   "preview-456",
	})
	if err != nil {
		fatalf(t, "Package error: %v", err)
	}

	if len(artifacts) != 3 {
		fatalf(t, "expected 3 artifacts, got %d", len(artifacts))
	}

	found := map[string]bool{}
	for _, a := range artifacts {
		found[a.Path] = true
		if a.Path == "dist/cloudflare/wrangler.toml" && !contains(a.Content, []byte("name = \"example\"")) {
			fatalf(t, "wrangler.toml missing project name: %s", string(a.Content))
		}
		if a.Path == "dist/cloudflare/wrangler.toml" && !contains(a.Content, []byte("binding = \"DISTLANG_KV\"")) {
			fatalf(t, "wrangler.toml missing kv binding: %s", string(a.Content))
		}
		if a.Path == "dist/cloudflare/wrangler.toml" && !contains(a.Content, []byte("id = \"namespace-123\"")) {
			fatalf(t, "wrangler.toml missing kv id: %s", string(a.Content))
		}
		if a.Path == "dist/cloudflare/Makefile" {
			if !contains(a.Content, []byte("npm install -g wrangler")) {
				fatalf(t, "makefile missing npm install line: %s", string(a.Content))
			}
			if !contains(a.Content, []byte("check-tools: deps")) {
				fatalf(t, "makefile missing deps dependency: %s", string(a.Content))
			}
			if !contains(a.Content, []byte("deps:")) {
				fatalf(t, "makefile missing deps target: %s", string(a.Content))
			}
		}
	}

	expected := []string{
		"dist/cloudflare/worker.js",
		"dist/cloudflare/wrangler.toml",
		"dist/cloudflare/Makefile",
	}
	for _, path := range expected {
		if !found[path] {
			fatalf(t, "missing artifact %s", path)
		}
	}
}

func fatalf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Fatalf(format, args...)
}

func contains(b, sub []byte) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(b); i++ {
		if string(b[i:i+len(sub)]) == string(sub) {
			return true
		}
	}
	return false
}
