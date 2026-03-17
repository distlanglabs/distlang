package v8

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSimpleAppProducesDualWorkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.js")
	src := `import { simpleApp } from "distlang/layers"; const handlerSet1 = { routes: { GET: { "/": async () => new Response("one") } } }; const handlerSet2 = { routes: { GET: { "/": async () => new Response("two") } } }; export default simpleApp.instantiate(handlerSet1, handlerSet2, {});`

	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	out, err := Build(path)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if len(out.Workers) != 2 {
		t.Fatalf("expected 2 workers, got %d", len(out.Workers))
	}
	if out.Workers[0].Name != "handlerSet1" {
		t.Fatalf("expected first worker handlerSet1, got %s", out.Workers[0].Name)
	}
	if out.Workers[1].Name != "handlerSet2" {
		t.Fatalf("expected second worker handlerSet2, got %s", out.Workers[1].Name)
	}
}
