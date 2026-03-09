package parse

import (
	"strings"
	"testing"
)

func TestToScriptKeepsESMForV8(t *testing.T) {
	src := `export default { async fetch(request) { return new Response("ok") } }`

	out, err := ToScript("index.js", src, FormatV8)
	if err != nil {
		t.Fatalf("ToScript error: %v", err)
	}

	if len(out) == 0 {
		t.Fatalf("expected output code")
	}

	if !strings.Contains(out, "export") {
		t.Fatalf("expected ESM export to remain for v8, got %s", out)
	}

	if strings.Contains(out, "distlangWorker") {
		t.Fatalf("unexpected global shim in v8 output: %s", out)
	}
}
