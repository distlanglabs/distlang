# distlang

## Current Status (Phase 0 POC)
- Go CLI with a Goja-backed runtime.
- ESM is transformed to Goja-friendly script via esbuild-based passes.
- `run` serves a Worker-style `default.fetch` over HTTP (strict worker mode).
- Commands: `build`, `target`, `deploy`, `run`, `debug`.

## Requirements
- Go 1.21+

## Quick Start
```bash
# build the CLI
go build -o ./bin/distlang ./cmd/distlang

# or use make
make build

# run from the example directory (default port 5656)
cd examples/helloworld
make run
# then open http://127.0.0.1:5656

# inspect compiler passes
make debug
```

## Commands
- `build <file>`: build platform artifacts (goja + cloudflare) into `dist/` and print a summary.
- `target init [--target=cloudflare] [--path=.]`: scaffold target files (including local env template) for a project/example.
- `deploy <file> [--target=cloudflare]`: build and deploy to a target platform (cloudflare for now).
- `run <file> [--port=N]`: start an HTTP server, load the worker, and route requests to `default.fetch` (strict worker mode; fails if `fetch` is missing).
- `debug <build|run> <file> [--passes=...]`: print pass outputs (`parse`, `ir`, `emit`); with `run`, also execute `fetch` once.

## Example Worker
```js
export default {
  async fetch(request, env, ctx) {
    return new Response("Hello Worker!");
  },
};
```
Run it locally (goja platform):
```bash
cd examples/helloworld
make run
```

Build artifacts (goja + cloudflare):
```bash
cd examples/helloworld
make build
# outputs summary and writes dist/ in the example directory

# Cloudflare Makefile helpers
make -C dist/cloudflare run      # wrangler dev
make -C dist/cloudflare publish  # wrangler deploy

# Distlang deploy helper (requires credentials)
../../bin/distlang target init --target=cloudflare --path=.
# then set values in targets/cloudflare/cloudflare.env
make deploy
```

## Known Limitations
- Minimal Web API shims (Request/Response/Headers are very limited).
- No hot reload; worker is loaded once at startup.
- Only Goja runtime backend; no WASM or other runtimes yet.
- Strict worker mode only; plain scripts are not executed by `run`.

## Roadmap
See [ROADMAP.md](ROADMAP.md) for vision, architecture, and upcoming phases.
