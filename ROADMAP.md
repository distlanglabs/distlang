# Roadmap

## Vision
Distlang is a capability-based framework for building portable serverless apps. The goal is a stable Distlang IR + Capability ABI that can target multiple backends (Goja/Node, workerd, Deno, WASI, future providers) with a consistent deploy/control plane and lock/version manager.

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
                           | invokes runtime formats       | today: goja, cloudflare
                           v                              v
         +--------------------+--------------------+--------------------+
         |                    |                    |                    |
         v                    v                    v                    v
   +-----------+       +-------------+       +-------------+      +-------------+
   | goja VM   |       | workerd     |       | Node runner |      | Deno runner |
   | (current) |       | (future)    |       | (future)    |      | (future)    |
   +-----------+       +-------------+       +-------------+      +-------------+

Current platform artifacts
  - goja: dist/goja/worker.js
  - cloudflare: dist/cloudflare/worker.js, wrangler.toml, Makefile
```

## Milestones

### Phase 0 (POC) — current
- Go CLI with Goja runtime.
- ESM→Goja transform (esbuild) and passes (`parse`, `ir`, `emit`).
- `run` serves Worker `default.fetch` over HTTP (strict worker mode).

### Phase 1 — runtime ergonomics
- Expand Web API shims (Request/Response/Headers, waitUntil queue, streaming bodies).
- Configurable request inputs for local dev (method, URL, headers, body), hot reload toggle.
- Improve debug outputs and pass descriptions; tighter error messages around exports.

### Phase 2 — compiler maturity
- Real emit stage (lowering to runtime-friendly JS or WASM-boundary-ready IR).
- IR stabilization and richer coverage of JS constructs.
- Test matrix hardening and fixtures for Worker semantics.

### Phase 3 — multi-backend
- Add secondary runtime backend (e.g., Node subprocess or workerd) behind the same ABI.
- Introduce deploy/control plane hooks and provider adapters.
- Begin WASM path exploration.

## Non-goals (now)
- Production-grade deploy plane and vendor adapters.
- Full capability surface (kv, http, log, etc.) beyond minimal stubs.
- Stable ABI/IR guarantees (until Phase 2).
