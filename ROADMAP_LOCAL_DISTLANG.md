# Local Distlang Roadmap

## Goal

Make `distlang local` provide an unauthenticated local Distlang experience at `/distlang`, backed by local SQLite, so users can record and explore Metrics without a Distlang account.

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
- Store local data under `~/.distlang/local/`
- Print the local URL and database path

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
- `/distlang/metrics/v1/api/v1/metadata`
- `/distlang/metrics/v1/api/v1/query`
- `/distlang/metrics/v1/api/v1/query_range`
- `/distlang/metrics/v1/metricsets/:metricSet/rows`

Use a fixed local owner scope, probably `userId = "local"`.

The local server should eventually become a unified local home for Distlang products, starting with Metrics and later including Agent Debugger.

## Shared Metrics Core

Extract reusable Metrics logic from `do-service` while keeping production behavior unchanged:

- Row schema helpers
- SQLite row store implementation
- Metadata generation
- Prom-style instant query
- Prom-style range query
- Dashboard-data query helpers

Production `do-service` keeps Durable Object-backed storage. Local Distlang uses SQLite-backed storage.

## Explorer UI

Do not export `distlang.com` as the local explorer UI. `distlang.com` should remain marketing and docs.

Instead:

- Reuse pieces from `dash/src/lib/metrics`
- Extract reusable Metrics Explorer components from `dash`
- Keep the hosted dashboard page as an auth-aware wrapper
- Add a local explorer wrapper with no auth and localStorage persistence

The local explorer should:

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

- Add this roadmap
- Link it from the main roadmap
- Add the local quickstart only when it is implemented or clearly marked upcoming

### Milestone 2: Metrics Core Extraction

- Extract reusable query and data logic from `do-service`
- Keep production behavior unchanged
- Add tests around extracted logic

### Milestone 3: Local SQLite Metrics API

- Add a local row store
- Add unauthenticated local metrics endpoints
- Verify writes, metadata, instant query, and range query

### Milestone 4: Local Explorer UI

- Extract reusable explorer components
- Add a local data source
- Serve at `/distlang/metrics`

### Milestone 5: CLI Integration

- Add `distlang local`
- Open the browser by default
- Print local URLs and database path

### Milestone 6: Docs And Install Polish

- Update install script final output
- Add `/docs/metrics/local`
- Add hosted/local mode chooser links
