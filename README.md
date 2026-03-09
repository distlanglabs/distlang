# distlang

# Plan

distlang will need to be split into two

1. distlang-js
   >> What is current right now. 
   So providers will be cloudflare-workers, netlify deno functions to start with.
   Get and echo app running with the control plane where the number of echos can be configured and the edge pipeline which responds with the echo.
2. distlang-wasm
  >> The wasm support will exist here. 

## Current Status (Phase 0 POC)
- Go CLI with a JavaScript worker build path and Cloudflare packaging.
- ESM is transformed into backend-ready JS via esbuild-based passes.
- `run` launches the local workerd runtime.
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
- `build <file>`: build backend artifacts (`v8`) and Cloudflare provider packaging into `dist/`.
- `target init [--target=cloudflare] [--path=.]`: scaffold target files (including local env template) for a project/example.
- `deploy <file> [--target=cloudflare]`: build and deploy to a target platform (cloudflare for now).
- `run <file> [--v8-port=N]`: build the V8 backend, then start local workerd.
- `debug <build|run> <file> [--passes=...]`: print pass outputs (`parse`, `ir`, `emit`); `debug run` now points you to `distlang run`.

## Example Worker
```js
export default {
  async fetch(request, env, ctx) {
    return new Response("Hello Worker!");
  },
};
```
Run it locally:
```bash
cd examples/helloworld
make run
```

Build artifacts (v8 + cloudflare package):
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
- `run` requires external `workerd`.
- No hot reload; worker processes are launched once per run invocation.
- Strict worker-style entrypoints are still assumed on the V8 path.

## Roadmap
See [ROADMAP.md](ROADMAP.md) for vision, architecture, and upcoming phases.
