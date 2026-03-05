# distlang

## Current Status (Phase 0 POC)
- Go CLI with a Goja-backed runtime.
- ESM is transformed to Goja-friendly script via esbuild-based passes.
- `run` serves a Worker-style `default.fetch` over HTTP (strict worker mode).
- Commands: `build`, `run`, `debug`.

## Requirements
- Go 1.21+

## Quick Start
```bash
# build the CLI
go build -o ./bin/distlang ./cmd/distlang

# or use make
make build

# run a worker locally (default port 5656)
./bin/distlang run examples/helloworld/index.js --port=5656
# then open http://127.0.0.1:5656

# inspect compiler passes
./bin/distlang debug build examples/helloworld/index.js --passes=parse,ir,emit
```

## Commands
- `build <file>`: build platform artifacts (goja + cloudflare) into `dist/` and print a summary.
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
./bin/distlang run examples/helloworld/index.js --port=5656
```

Build artifacts (goja + cloudflare):
```bash
./bin/distlang build examples/helloworld/index.js
# outputs summary and writes dist/

# Cloudflare Makefile helpers
make -C dist/cloudflare run      # wrangler dev
make -C dist/cloudflare publish  # wrangler deploy
```

## Known Limitations
- Minimal Web API shims (Request/Response/Headers are very limited).
- No hot reload; worker is loaded once at startup.
- Only Goja runtime backend; no WASM or other runtimes yet.
- Strict worker mode only; plain scripts are not executed by `run`.

## Roadmap
See [ROADMAP.md](ROADMAP.md) for vision, architecture, and upcoming phases.
