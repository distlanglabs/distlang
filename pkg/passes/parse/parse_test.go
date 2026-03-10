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

func TestToScriptGeneratesVisibleCoreHelper(t *testing.T) {
	src := `import { ObjectDB } from "distlang/core"; export default { async fetch(request, env, ctx) { return Response.json(await ObjectDB.get("hello")) } }`

	res, err := ToScriptWithOptions("index.js", src, Options{Format: FormatV8})
	if err != nil {
		t.Fatalf("ToScriptWithOptions error: %v", err)
	}

	if len(res.Artifacts) != 1 {
		t.Fatalf("expected 1 generated artifact, got %d", len(res.Artifacts))
	}
	if res.Artifacts[0].Path != "generated/distlang/core/index.js" {
		t.Fatalf("unexpected generated path: %s", res.Artifacts[0].Path)
	}
	if !strings.Contains(string(res.Artifacts[0].Content), "export const ObjectDB") {
		t.Fatalf("generated helper missing ObjectDB export: %s", string(res.Artifacts[0].Content))
	}
	if !strings.Contains(res.Code, "wrapWorkerWithObjectDB") {
		t.Fatalf("expected wrapped default export in emitted code: %s", res.Code)
	}
	if strings.Contains(res.Code, `from "distlang/core"`) || strings.Contains(res.Code, `from 'distlang/core'`) {
		t.Fatalf("expected distlang/core to be bundled away: %s", res.Code)
	}
}
