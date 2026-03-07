# distlang

## Current Status (Phase 0 POC)
- Go CLI with backend-oriented builds for V8 and Wasm.
- ESM is transformed into backend-ready JS via esbuild-based passes.
- `run` now prepares both local runtime paths: workerd and wasmtime.
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
- `build <file>`: build backend artifacts (`v8`, `wasm`) and Cloudflare provider packaging into `dist/`.
- `target init [--target=cloudflare] [--path=.]`: scaffold target files (including local env template) for a project/example.
- `deploy <file> [--target=cloudflare]`: build and deploy to a target platform (cloudflare for now).
- `run <file> [--v8-port=N] [--wasm-port=N]`: build both backends, then start local workerd and wasmtime runtimes.
- `debug <build|run> <file> [--passes=...]`: print pass outputs (`parse`, `ir`, `emit`); `debug run` now points you to `distlang run`.

## Example Worker
```js
export default {
  async fetch(request, env, ctx) {
    return new Response("Hello Worker!");
  },
};
```
Run it locally (dual-runtime mode):
```bash
cd examples/helloworld
make run
```

Build artifacts (v8 + wasm + cloudflare package):
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
- Wasm output is still a backend workspace placeholder until Distlang lowers IR to runnable Wasm artifacts.
- `run` requires external `workerd` and `wasmtime` binaries.
- No hot reload; worker processes are launched once per run invocation.
- Strict worker-style entrypoints are still assumed on the V8 path.

## Roadmap
See [ROADMAP.md](ROADMAP.md) for vision, architecture, and upcoming phases.
