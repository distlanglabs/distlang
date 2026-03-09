# Roadmap

## Vision
Distlang is a capability-based framework for building portable serverless apps. The current focus is a stable JavaScript edge app model with provider packaging layered on top.

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
                           | invokes backend formats       | today: v8, cloudflare package
                            v                              v
         +--------------------+--------------------+--------------------+
         |                    |                    |                    |
         v                    v                    v                    v
   +-----------+       +-------------+       +-------------+      +-------------+
   | workerd   |       | netlify     |       | Node runner |      | Deno runner |
   | (local)   |       | (future)    |       | (future)    |      | (future)    |
   +-----------+       +-------------+       +-------------+      +-------------+

Current platform artifacts
  - v8: dist/v8/worker.js
  - cloudflare: dist/cloudflare/worker.js, wrangler.toml, Makefile
```

## Milestones

### Phase 0 (POC) — current
- Go CLI with a V8 build path and Cloudflare packaging.
- ESM transform (esbuild) and passes (`parse`, `ir`, `emit`).
- `run` launches the local workerd entrypoint.

### Phase 1 — runtime ergonomics
- Expand Web API shims (Request/Response/Headers, waitUntil queue, streaming bodies).
- Configurable request inputs for local dev (method, URL, headers, body), hot reload toggle.
- Improve debug outputs and pass descriptions; tighter error messages around exports.

### Phase 2 — compiler maturity
- Real emit stage (lowering to runtime-friendly JS).
- IR stabilization and richer coverage of JS constructs.
- Test matrix hardening and fixtures for Worker semantics.

### Phase 3 — multi-backend
- Introduce deploy/control plane hooks and provider adapters.
- Expand provider packaging beyond Cloudflare.

## Non-goals (now)
- Production-grade deploy plane and vendor adapters.
- Full capability surface (kv, http, log, etc.) beyond minimal stubs.
- Stable ABI/IR guarantees (until Phase 2).
