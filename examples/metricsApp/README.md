# metricsApp

This example shows `simpleApp` with `ObjectDB` plus flush-based metrics.

## Build

From the `distlang` repo root:

```bash
go build -o bin/distlang ./cmd/distlang
./bin/distlang build index.js
```

Or from this directory:

```bash
../../bin/distlang build index.js
```

## Verify locally

Run the runtime smoke test from the `distlang` repo root:

```bash
node test/runtime/helpers_metrics_smoke.mjs
```

That test builds the CLI, rebuilds this example, exercises mock mode, exercises live mode against the local mock helpers server, and verifies public metrics reads.

## What to check manually

- `POST /echo/config` stores config and records labeled counter + histogram data
- `GET /echo/:text` reads config and records labeled counter + histogram data
- `POST /metrics/query` returns:
  - `series` from the public Prometheus-style `query_range` API
  - `metadata` for `edgeReqCount`
  - discovered label names

## Metric definitions

This example uses object-form metric definitions with:

- `kind`
- `description`
- `unit`
- `labels`

and writes labels inline, for example:

```js
metrics.edgeReqCount.inc({ route: "/echo/:text", method: "GET", status: "200" });
metrics.dbCallLatency.observe(12, { operation: "get" });
```
