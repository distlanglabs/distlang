# Local Distlang Roadmap

## Goal

Make `distlang local` provide an unauthenticated local Distlang experience at `/distlang`, backed by local SQLite owned by the installed `distlang` tool, so users can record and explore Metrics without a Distlang account.

## Current Status

Local Distlang is implemented as a single-process local Metrics stack inside the `distlang` binary:

- `distlang local` starts a local server on `127.0.0.1:4817` by default
- The local home is served at `http://127.0.0.1:4817/distlang`
- The local Metrics explorer is served at `http://127.0.0.1:4817/distlang/metrics`
- SQLite is the default local Metrics store
- The default database path is `~/.distlang/local/metrics.db`
- `--db=PATH` can point local storage at a specific SQLite file
- `--memory` keeps the in-memory test/development backend available
- Users do not need to install SQLite separately
- Local Metrics APIs are unauthenticated

The current local explorer is embedded in the CLI server. It supports:

- Metadata browsing
- Prom-style instant queries
- SQL queries against the local SQLite metrics tables
- Table and raw JSON result views for SQL output
- SQL shortcut buttons for common schema/data inspection

The longer-term dashboard component extraction remains future UX polish, not a prerequisite for local Metrics to function.

## UX Decision

Hosted Distlang remains the production and team path:

- API: `https://api.distlang.com`
- UI: `https://dash.distlang.com`
- Requires a Distlang account and API token
- Current `/docs/metrics/quickstart` stays hosted-first

Local Distlang becomes the open-source, development, and private-inspection path:

- API and UI: `http://127.0.0.1:4817/distlang`
- No account required
- No API token required
- Data stored locally in SQLite

## Install Flow

The installer should continue to install only the `distlang` binary. It should not auto-start a local service.

After install, the final output should point users to both paths:

- Local: `distlang local`
- Hosted: `distlang helpers login`

Example installer ending:

```text
Installed distlang.

Start local Distlang, no account required:
  distlang local

Use hosted Distlang:
  distlang helpers login

Docs:
  https://distlang.com/docs
```

## CLI Work

Add a top-level local command:

```bash
distlang local
```

Initial behavior:

- Start a local server on `127.0.0.1:4817`
- Open `http://127.0.0.1:4817/distlang`
- Create or open the local database automatically
- Store local data under `~/.distlang/local/`
- Print the local URL and database path
- Use SQLite by default, unless `--memory` is supplied
- Accept `--db=PATH` for an explicit local SQLite file

Potential later subcommands:

```bash
distlang local start
distlang local status
distlang local open
distlang local stop
```

Avoid `distlang serve` for this UX because `helpers serve` already means the helper mock server, and `run` already means local app runtime.

## Local Server

Serve local Distlang pages and APIs without auth:

- `/distlang`
- `/distlang/metrics`
- `/distlang/metrics/v1/capabilities`
- `/distlang/metrics/v1/sql`
- `/distlang/metrics/v1/api/v1/metadata`
- `/distlang/metrics/v1/api/v1/query`
- `/distlang/metrics/v1/api/v1/query_range`
- `/distlang/metrics/v1/metricsets/:metricSet/rows`

Use a fixed local owner scope, probably `userId = "local"`.

The local server should run inside the `distlang local` process. Users should not need to install SQLite separately, run a database daemon, or start another service.

The local server should eventually become a unified local home for Distlang products, starting with Metrics and later including Agent Debugger.

## Query Backend Contract

Local Metrics now uses a backend-agnostic query contract so the explorer can work with local SQLite today and a future authenticated remote SQL backend later.

The contract includes:

- Capabilities discovery
- Metadata requests
- Instant query requests
- Range query requests
- SQL query requests

The `/distlang/metrics/v1/capabilities` endpoint tells the explorer which features are available for the active backend. The SQLite backend reports SQL support; the in-memory backend keeps SQL disabled.

## Local Storage Model

SQLite should be an implementation detail of the installed `distlang` tool.

Users should not need to install SQLite, run a separate database, or start a separate service. `distlang local` should create and manage the database file under `~/.distlang/local/`.

The local Metrics API should depend on a small store interface so the HTTP server is not coupled directly to SQLite. The first implementation is SQLite-backed local storage; tests may use an in-memory implementation.

Current SQLite tables:

- `metric_sets`
- `metric_definitions`
- `metric_rows`
- `metric_row_values`

The SQLite store persists metric definitions and emitted metric rows. Prom-style local queries run through the shared query backend, while raw SQL queries execute against the same SQLite database with read-only guardrails.

## Local SQL Explorer

The local explorer supports read-only SQL through `/distlang/metrics/v1/sql`.

Supported SQL behavior:

- `SELECT` and `WITH` queries are allowed
- Multiple SQL statements are rejected
- Write/schema/database-management tokens are blocked
- Result rows are capped by a server-side maximum
- Query execution uses a short timeout

Schema discovery commands are handled as safe meta-commands:

- `show tables;`
- `.tables`
- `describe <table>;`
- `describe table <table>;`
- `desc <table>;`
- `.schema <table>`

The explorer renders SQL results as a readable table by default and keeps a raw JSON view for debugging the exact API payload.

## Hosted Explorer Plan

The local explorer should continue to start the same way:

```bash
distlang local
```

From that same browser UI, users should be able to switch between local SQLite data and hosted account data. The hosted path should feel like the local path with authentication added, not like a separate product or separate explorer.

Public explorer-facing endpoints should remain metric-set agnostic:

- `GET /distlang/metrics/v1/capabilities`
- `GET /distlang/metrics/v1/api/v1/metadata`
- `GET /distlang/metrics/v1/api/v1/query`
- `GET /distlang/metrics/v1/api/v1/query_range`
- `POST /distlang/metrics/v1/sql`

The local process may proxy hosted calls through source-specific local routes to avoid exposing tokens to browser JavaScript and to avoid CORS issues. The remote API itself should still mirror the local explorer contract and require normal hosted authentication.

Metric sets should be abstracted as data, not as required route segments in the explorer contract. The explorer and queries can still expose metric sets through metadata, labels, SQL predicates, or UI selection:

- Prom-style queries use labels such as `metricSet="my-app"`
- SQL can query `metric_sets` or filter with `where metric_set = 'my-app'`
- Explorer source state can track the selected metric set without changing endpoint names

Hosted implementation should use a query gateway over Durable Object SQLite storage:

- The gateway authenticates the user using normal hosted auth
- The gateway discovers metric sets available to the account
- The gateway routes narrow queries to one `MetricsTimeBucketDO` when possible
- The gateway can fan out simple metadata/sample queries across metric sets where needed
- Each `MetricsTimeBucketDO` keeps owning its SQLite database for one user and metric set
- SQL responses must use the same `columns`, `rows`, and `stats` shape as local SQLite

The Durable Object SQL support should be implemented as a storage/query plugin, not hardcoded into the explorer:

- `LocalSQLiteQueryBackend` for local CLI SQLite
- `DurableObjectSQLiteQueryBackend` for hosted Metrics Durable Objects
- Future backends can implement the same capabilities/query/SQL contract

Hosted SQL should expose a virtual schema where useful. The physical Durable Object tables may differ from local tables, but explorer-facing capabilities should describe what users can query. A hosted `metric_rows` view should include a `metric_set` column even when that value is synthetic from the routed Durable Object.

Initial hosted SQL should prioritize practical read-only queries:

- `show tables;`
- `describe <table>;`
- `select * from metric_sets`
- `select * from metric_rows where metric_set = 'my-app' limit 20`
- simple aggregate queries with a metric-set filter
- limited cross-metric-set metadata and sample-row queries when the gateway can safely merge results

Full arbitrary cross-Durable Object SQL federation is explicitly later work. If a query cannot be safely routed or merged, the gateway should return a clear error suggesting a `metric_set` filter.

Hosted mode should keep the same permissions as existing hosted read/query access. If an authenticated user can read/query a metric set, they can run read-only SQL for that metric set. There should be no separate SQL role or extra permission layer. Safety guardrails still apply because they protect storage integrity and service health, not because they are a separate permission model.

Implementation phases:

1. Make the explorer use a configurable API base instead of hardcoded local paths.
2. Add a source switcher for Local SQLite and Hosted Distlang inside the existing explorer.
3. Generate SQL shortcut examples from `capabilities.schema` instead of hardcoded table names.
4. Add local hosted-proxy routes in `distlang local` that use existing CLI auth state.
5. Add hosted authenticated explorer-contract endpoints that mirror the local endpoint names.
6. Add Durable Object capabilities and read-only SQL execution behind the hosted query gateway.
7. Add tests proving local and hosted return the same response shapes for capabilities, metadata, Prom-style query, and SQL table results.

## Shared Metrics Core

Extract reusable Metrics logic from `do-service` while keeping production behavior unchanged:

- Row schema helpers
- Local Metrics store abstraction
- SQLite-backed local store implementation inside `distlang`
- Metadata generation
- Prom-style instant query
- Prom-style range query
- Dashboard-data query helpers

Production `do-service` keeps Durable Object-backed storage. Local Distlang uses SQLite-backed storage through the installed `distlang` binary.

## Explorer UI

Do not export `distlang.com` as the local explorer UI. `distlang.com` should remain marketing and docs.

Long term:

- Reuse pieces from `dash/src/lib/metrics`
- Extract reusable Metrics Explorer components from `dash`
- Keep the hosted dashboard page as an auth-aware wrapper
- Add a local explorer wrapper with no auth and localStorage persistence

Current implementation:

- Embedded local explorer served by `distlang local`
- Metadata list from local Metrics metadata
- Prom-style instant query input
- SQL query textarea
- SQL shortcuts for schema and sample row inspection
- Table/raw JSON SQL result views

The future extracted explorer should:

- List metric sets from metadata
- Let the user choose a metric set
- Auto-generate panels from metric definitions
- Save layout and time preferences locally
- Disable hosted-only features such as billing, support view, ObjectDB dashboard persistence, and AI suggestions

## Docs

Keep the current hosted Metrics quickstart as the main hosted path:

- `/docs/metrics/quickstart`

Add a short note near the top:

```md
This quickstart sends metrics to hosted Distlang Metrics at `api.distlang.com` and shows them in `dash.distlang.com`.

If you want to run Distlang locally without an account or API token, use the [Local Metrics Quickstart](/docs/metrics/local/) instead.
```

Add a separate local page:

- `/docs/metrics/local`

The local page should cover:

1. Install the CLI
2. Run `distlang local`
3. Install `@distlang/client`
4. Send metrics to the local base URL
5. Open the local explorer

Do not publish the local quickstart as production docs until `distlang local` and local Metrics APIs exist, unless the page is clearly marked upcoming.

## JavaScript Client

Short term, local examples can configure the base URL:

```js
const client = createDistlangClient({
  storeBaseURL: "http://127.0.0.1:4817/distlang",
});
```

If the client still requires an access token, examples may need a temporary compatibility value:

```js
const metrics = client.metrics.createRecorder({
  accessToken: "local",
  metricSet: "quickstart-app",
  definitions: {
    requestCount: "counter",
    latencyMs: "histogram",
  },
});
```

Better future API:

- Allow no token for local mode
- Consider `mode: "local"`
- Make recorder `accessToken` optional when the configured base URL is local

## Milestones

### Milestone 1: Roadmap And Docs Skeleton

Status: implemented.

- Add this roadmap
- Link it from the main roadmap
- Add the local quickstart only when it is implemented or clearly marked upcoming

### Milestone 2: Metrics Core Extraction

Status: partially implemented.

- Extract reusable query and data logic from `do-service`
- Keep production behavior unchanged
- Add tests around extracted logic

### Milestone 3: Local SQLite Metrics API

Status: implemented for local Metrics write, metadata, instant query, range-query API surface, and SQL query support.

- Add a local Metrics store abstraction inside `distlang`
- Back the first local store with SQLite owned by the installed `distlang` binary
- Store database files under `~/.distlang/local/`
- Add unauthenticated local metrics endpoints
- Verify writes, metadata, instant query, and range query
- Add SQL capabilities and read-only SQL endpoint
- Add schema discovery meta-commands

### Milestone 4: Local Explorer UI

Status: implemented as an embedded local explorer; dashboard component extraction remains future polish.

- Extract reusable explorer components
- Add a local data source
- Serve at `/distlang/metrics`
- Add SQL table/raw views and schema shortcuts

### Milestone 5: CLI Integration

Status: implemented.

- Add `distlang local`
- Open the browser by default
- Print local URLs and database path
- Add `--db=PATH` and `--memory`

### Milestone 6: Docs And Install Polish

Status: pending.

- Update install script final output
- Add `/docs/metrics/local`
- Add hosted/local mode chooser links

### Milestone 7: Hosted Data Source In Local Explorer

Status: planned.

- Refactor explorer JavaScript to use configurable endpoint bases
- Add a Local SQLite / Hosted Distlang source switcher
- Proxy hosted requests through the local `distlang local` process using CLI auth
- Keep explorer-facing endpoint names aligned between local and hosted
- Abstract metric sets as metadata/query data instead of route structure
- Add hosted capabilities and SQL support through a Durable Object SQLite query backend
- Preserve identical SQL table/raw result rendering for local and hosted responses
