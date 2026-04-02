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
	src := `import { InMemDB } from "distlang/core"; export default { async fetch(request, env, ctx) { return Response.json(await InMemDB.get("hello")) } }`

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
	if !strings.Contains(string(res.Artifacts[0].Content), "export const InMemDB") {
		t.Fatalf("generated helper missing InMemDB export: %s", string(res.Artifacts[0].Content))
	}
	if !strings.Contains(res.Code, "wrapWorkerWithInMemDB") {
		t.Fatalf("expected wrapped default export in emitted code: %s", res.Code)
	}
	if strings.Contains(res.Code, `from "distlang/core"`) || strings.Contains(res.Code, `from 'distlang/core'`) {
		t.Fatalf("expected distlang/core to be bundled away: %s", res.Code)
	}
}

func TestToScriptGeneratesDistlangHelpersModule(t *testing.T) {
	src := `import { helpers } from "distlang"; export default { async fetch(request) { return Response.json(await helpers.ObjectDB.status()) } }`

	res, err := ToScriptWithOptions("index.js", src, Options{Format: FormatV8})
	if err != nil {
		t.Fatalf("ToScriptWithOptions error: %v", err)
	}

	if len(res.Artifacts) < 2 {
		t.Fatalf("expected at least 2 generated artifacts, got %d", len(res.Artifacts))
	}
	artifactByPath := map[string]string{}
	for _, artifact := range res.Artifacts {
		artifactByPath[artifact.Path] = string(artifact.Content)
	}
	if _, ok := artifactByPath["generated/distlang/core/index.js"]; !ok {
		t.Fatalf("missing generated core helper: %#v", artifactByPath)
	}
	if _, ok := artifactByPath["generated/distlang/index.js"]; !ok {
		t.Fatalf("missing generated helpers index: %#v", artifactByPath)
	}
	if !strings.Contains(artifactByPath["generated/distlang/index.js"], "export const helpers") {
		t.Fatalf("generated helper missing helpers export: %s", artifactByPath["generated/distlang/index.js"])
	}
	if !strings.Contains(res.Code, "wrapWorkerWithHelpers") {
		t.Fatalf("expected helpers wrapper in emitted code: %s", res.Code)
	}
	if strings.Contains(res.Code, `from "distlang"`) || strings.Contains(res.Code, `from 'distlang'`) {
		t.Fatalf("expected distlang module to be bundled away: %s", res.Code)
	}
}

func TestToScriptGeneratesLayersModule(t *testing.T) {
	src := `import { simpleApp } from "distlang/layers"; const handlerSet1 = { routes: { GET: { "/": async ({ req, db }) => new Response("one") } } }; const handlerSet2 = { routes: { GET: { "/": async ({ req, db }) => new Response("two") } } }; export default simpleApp.instantiate(handlerSet1, handlerSet2, { get: async () => null, put: async () => null });`

	res, err := ToScriptWithOptions("index.js", src, Options{Format: FormatV8})
	if err != nil {
		t.Fatalf("ToScriptWithOptions error: %v", err)
	}

	if len(res.Artifacts) != 1 {
		t.Fatalf("expected 1 generated artifact, got %d", len(res.Artifacts))
	}
	if res.Artifacts[0].Path != "generated/distlang/layers/index.js" {
		t.Fatalf("unexpected generated path: %s", res.Artifacts[0].Path)
	}
	if !strings.Contains(string(res.Artifacts[0].Content), "export const simpleApp") {
		t.Fatalf("generated helper missing simpleApp export: %s", string(res.Artifacts[0].Content))
	}
	if !res.UsesLayers {
		t.Fatalf("expected UsesLayers to be true")
	}
	if strings.Contains(res.Code, `from "distlang/layers"`) || strings.Contains(res.Code, "from 'distlang/layers'") {
		t.Fatalf("expected distlang/layers module to be bundled away: %s", res.Code)
	}
}

func TestToScriptGeneratesAppModule(t *testing.T) {
	src := `import { app } from "distlang/app"; export default app({ state: { dbs: { ObjectDB: { get: async () => null, put: async () => null, buckets: { create: async () => null }, keys: { list: async () => [] } } } }, compute: { handlers: { routes: { GET: { "/": async ({ req, state, params }) => new Response("ok") } } } } });`

	res, err := ToScriptWithOptions("index.js", src, Options{Format: FormatV8})
	if err != nil {
		t.Fatalf("ToScriptWithOptions error: %v", err)
	}

	if len(res.Artifacts) != 1 {
		t.Fatalf("expected 1 generated artifact, got %d", len(res.Artifacts))
	}
	if res.Artifacts[0].Path != "generated/distlang/app/index.js" {
		t.Fatalf("unexpected generated path: %s", res.Artifacts[0].Path)
	}
	if !strings.Contains(string(res.Artifacts[0].Content), "export function app") {
		t.Fatalf("generated helper missing app export: %s", string(res.Artifacts[0].Content))
	}
	if strings.Contains(res.Code, `from "distlang/app"`) || strings.Contains(res.Code, "from 'distlang/app'") {
		t.Fatalf("expected distlang/app module to be bundled away: %s", res.Code)
	}
}

func TestToScriptChainsDistlangWrappers(t *testing.T) {
	src := `import { InMemDB } from "distlang/core"; import { helpers } from "distlang"; export default { async fetch(request) { await InMemDB.put("a", 1); return Response.json(await helpers.ObjectDB.status()) } }`

	res, err := ToScriptWithOptions("index.js", src, Options{Format: FormatV8})
	if err != nil {
		t.Fatalf("ToScriptWithOptions error: %v", err)
	}

	if !strings.Contains(res.Code, "wrapWorkerWithHelpers(wrapWorkerWithInMemDB(") {
		t.Fatalf("expected chained wrappers in emitted code: %s", res.Code)
	}
}
