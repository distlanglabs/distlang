package parse

import (
	"strings"
	"testing"
)

func TestToScriptConvertsExportDefault(t *testing.T) {
	src := `export default { async fetch(request) { return new Response("ok") } }`

	out, err := ToScript("index.js", src, FormatGoja)
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

func TestToScriptKeepsESMForCloudflare(t *testing.T) {
	src := `export default { async fetch(request) { return new Response("ok") } }`

	out, err := ToScript("index.js", src, FormatCloudflare)
	if err != nil {
		t.Fatalf("ToScript error: %v", err)
	}

	if !strings.Contains(out, "export") {
		t.Fatalf("expected ESM export to remain for cloudflare, got %s", out)
	}
	if strings.Contains(out, "distlangWorker") {
		t.Fatalf("unexpected goja global in cloudflare output: %s", out)
	}
}
