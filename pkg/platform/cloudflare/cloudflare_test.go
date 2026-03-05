package cloudflare

import "testing"

func TestRenderProducesArtifacts(t *testing.T) {
	artifacts, err := Render("console.log('hi')", Context{ProjectName: "example"})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if len(artifacts) != 3 {
		t.Fatalf("expected 3 artifacts, got %d", len(artifacts))
	}

	found := map[string]bool{}
	for _, a := range artifacts {
		found[a.Path] = true
		if a.Path == "dist/cloudflare/wrangler.toml" && !contains(a.Content, []byte("name = \"example\"")) {
			t.Fatalf("wrangler.toml missing project name: %s", string(a.Content))
		}
		if a.Path == "dist/cloudflare/Makefile" {
			if !contains(a.Content, []byte("npm install -g wrangler")) {
				t.Fatalf("makefile missing npm install line: %s", string(a.Content))
			}
			if !contains(a.Content, []byte("check-tools: deps")) {
				t.Fatalf("makefile missing deps dependency: %s", string(a.Content))
			}
			if !contains(a.Content, []byte("deps:")) {
				t.Fatalf("makefile missing deps target: %s", string(a.Content))
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
			t.Fatalf("missing artifact %s", path)
		}
	}
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
