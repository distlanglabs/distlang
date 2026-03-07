# Roadmap

## Vision
Distlang is a capability-based framework for building portable serverless apps. The goal is a stable Distlang IR + Capability ABI that can target execution backends (V8 isolates, Wasm hosts, future container paths) with provider packaging layered on top.

## Architecture (current POC + future-facing)
```
                            +----------------------+
                            |      distlang CLI    |
                            |   (Go orchestrator)  |
                            +----------+-----------+
                                       |
                     +-----------------+-----------------+
                     |                                   |
                     v                                   v
           +---------------------+               +----------------------+
           |  Compiler/Planner   |               |  Platform Integrator |
           |      (Go)           |               |      (Go, future)    |
           +----------+----------+               +----------+-----------+
                      |                                  |
                      | produces                          | renders
                      v                                  v
             +----------------------------+   +-------------------------------+
             | Distlang IR + Capability   |   | Platform artifacts / configs  |
             | ABI (future-stable)        |   | (dist/*)                      |
             +-------------+--------------+   +-------------------------------+
                           |                              |
                           | invokes backend formats       | today: v8, wasm, cloudflare package
                           v                              v
         +--------------------+--------------------+--------------------+
         |                    |                    |                    |
         v                    v                    v                    v
   +-----------+       +-------------+       +-------------+      +-------------+
   | workerd   |       | wasmtime    |       | Node runner |      | Deno runner |
   | (local)   |       | (local)     |       | (future)    |      | (future)    |
   +-----------+       +-------------+       +-------------+      +-------------+

Current platform artifacts
  - v8: dist/v8/worker.js
  - wasm: dist/wasm/*
  - cloudflare: dist/cloudflare/worker.js, wrangler.toml, Makefile
```

## Milestones

### Phase 0 (POC) — current
- Go CLI with backend builds for V8 and Wasm.
- ESM transform (esbuild) and passes (`parse`, `ir`, `emit`).
- `run` launches local workerd and wasmtime entrypoints.

### Phase 1 — runtime ergonomics
- Expand Web API shims (Request/Response/Headers, waitUntil queue, streaming bodies).
- Configurable request inputs for local dev (method, URL, headers, body), hot reload toggle.
- Improve debug outputs and pass descriptions; tighter error messages around exports.

### Phase 2 — compiler maturity
- Real emit stage (lowering to runtime-friendly JS or WASM-boundary-ready IR).
- IR stabilization and richer coverage of JS constructs.
- Test matrix hardening and fixtures for Worker semantics.

### Phase 3 — multi-backend
- Mature the V8 and Wasm backends into runnable parity.
- Introduce deploy/control plane hooks and provider adapters.
- Expand provider packaging beyond Cloudflare.

## Non-goals (now)
- Production-grade deploy plane and vendor adapters.
- Full capability surface (kv, http, log, etc.) beyond minimal stubs.
- Stable ABI/IR guarantees (until Phase 2).
