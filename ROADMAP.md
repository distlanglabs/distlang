# Roadmap

## Vision
Distlang is evolving toward a portable app platform built around reusable helpers and hosted data services.

The compiler and CLI remain important, but the near-term focus is making helpers and store-backed capabilities reliable enough to stand on their own, both inside distlang apps and in non-distlang apps.

## Architecture Direction
```
+---------------------+
|  distlang CLI       |
|  compiler + deploy  |
+----------+----------+
           |
           v
+---------------------+
| helpers layer       |
| ObjectDB, metrics,  |
| app-facing APIs     |
+----------+----------+
           |
           v
+---------------------+
| store services      |
| objectdb, analytics,|
| auth-backed access  |
+----------+----------+
           |
           v
+---------------------+
| runtimes/providers  |
| local workerd,      |
| Cloudflare, hosted  |
+---------------------+
```

distlang currently spans three closely related pieces:

- CLI/compiler: builds JavaScript worker apps into runnable and deployable artifacts.
- helpers layer: app-facing capabilities such as ObjectDB and metrics.
- store services: hosted APIs that back ObjectDB, analytics, and future helper capabilities.

Near-term roadmap work prioritizes helpers and store over broader compiler or provider expansion.

## Current Position

- Phase 0 is largely complete.
- Phase 1 is actively in progress and already has significant implementation.
- Phase 2 has not been completed, but some prerequisites already exist.
- Phase 3 is partially underway through auth, dashboard, and store surface work.

Evidence already in the workspace:

- `distlang` has generated helper modules, ObjectDB and metrics examples, local mock/live flows, hosted deploys, and smoke tests.
- `metrics-service` already has a concrete public/internal route plan plus external deployment-and-validation tooling for metrics visibility and latency.
- `user-auth` already supports CLI login, refresh tokens, and service tokens for store-backed access.
- `dash` already consumes deployments, ObjectDB, auth, and metrics data in a real UI.
- `cloudflare-analytics-debug` already exists to inspect Analytics Engine ingestion and query timing.

## Milestones

### Phase 0 (POC) — mostly complete
- Go CLI with a V8 build path and Cloudflare packaging.
- Local `workerd` runtime flow.
- Generated helper modules for `distlang/core`, `distlang`, `distlang/app`, and `distlang/layers`.
- Authenticated store access through `helpers`.
- Hosted deploy flow for Distlang-managed Cloudflare hosting.

### Phase 1 — stable metrics in store
Already in place:

- Metrics helper APIs in distlang apps.
- Mock and live metrics flows.
- Metrics example apps and runtime smoke coverage.
- Public metrics read and query endpoints.
- Dashboard and external test tooling around metrics visibility.

Remaining work:

- Lock down the stable metrics API and expected app-facing contracts.
- Harden retention, latency, consistency, and failure semantics.
- Validate end-to-end metrics visibility with repeated hosted runs and dashboard timing checks.
- Decide what stays in `store` vs what later moves behind `metrics-service`.
- Define clear exit criteria for correctness and operational stability.

### Phase 2 — helpers with or without distlang
- Make helpers usable with or without distlang.
- Support the same helper APIs in distlang-built apps and plain JavaScript worker apps.
- Reduce coupling between helper APIs and build-time code generation where possible.
- Clarify packaging and runtime expectations for standalone helper usage.
- Document the standalone helper model with examples.

Groundwork already exists:

- Auth and service-token flows already exist.
- Public helper-facing store routes already exist.
- Current helper APIs already work inside distlang-built apps.

What is still missing is a clean standalone helper packaging and runtime story.

### Phase 3 — store maturity
- Strengthen ObjectDB and analytics operational semantics.
- Improve auth, session, and service-token ergonomics for helper users.
- Add clearer service contracts, limits, and debugging flows.
- Expand test coverage around store failure modes and consistency expectations.
- Keep `store` as the stable public gateway even if internal metrics backends split later.

### Phase 4 — compiler and runtime maturity
- Improve local runtime ergonomics and debug outputs.
- Continue IR and emit-stage maturation where it directly supports helper portability.
- Keep backend and provider expansion secondary to helper and store stability.

## Non-goals (now)
- Broad multi-provider expansion before helper and store APIs are stable.
- Premature ABI or IR guarantees beyond what helper portability requires.
- Large capability-surface expansion before ObjectDB and metrics are solid.
