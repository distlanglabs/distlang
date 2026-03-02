package parse

import (
	"strings"
	"testing"
)

func TestToScriptConvertsExportDefault(t *testing.T) {
	src := `export default { async fetch(request) { return new Response("ok") } }`

	out, err := ToScript("index.js", src)
	if err != nil {
		t.Fatalf("ToScript error: %v", err)
	}

	if len(out) == 0 {
		t.Fatalf("expected output code")
	}

	if strings.Contains(out, "export default") {
		t.Fatalf("expected transformed code without ESM export, got %s", out)
	}

	if !strings.Contains(out, "distlangWorker") {
		t.Fatalf("expected global name distlangWorker in output, got %s", out)
	}
}
