# distlang

Website: [distlang.com](https://distlang.com)

API docs: [api.distlang.com/docs](https://api.distlang.com/docs)

## Install

Stable release (Linux/macOS):
```bash
curl -fsSL https://distlang.com/install | bash
```

Latest from `main` for testing (Linux/macOS):
```bash
curl -fsSL https://distlang.com/install-main | bash
```

Windows PowerShell:
```powershell
irm https://distlang.com/install.ps1 | iex
```

Verify install:
```bash
distlang --version
```

By default the installer places `distlang` in a user-local bin directory (`~/.local/bin` on Linux/macOS, `%LOCALAPPDATA%\distlang\bin` on Windows).

## Requirements
- Go 1.21+

## Quick Start
```bash
# install the CLI
curl -fsSL https://distlang.com/install | bash

# or test the latest main build
curl -fsSL https://distlang.com/install-main | bash

# build an app
distlang build examples/app-echo/index.js

# run it locally (default port 5656)
distlang run examples/app-echo/index.js

# deploy it through Distlang hosting
distlang deploy examples/app-echo/index.js
```

## High-Level Overview

`distlang` is a Go CLI for building JavaScript worker apps into runnable and deployable distributed services.

- It compiles app source into backend-ready V8 worker code.
- It generates helper modules when apps import `distlang/core`, `distlang`, `distlang/app`, or `distlang/layers`.
- It runs workers locally with `workerd`.
- It packages apps for Cloudflare and can also deploy through Distlang-managed hosting.

```text
app source (index.js)
        |
        v
distlang CLI
(build / run / deploy / debug)
        |
        v
compile pipeline
source -> parse/bundle -> emitted JS
        |
        +--> generated helper modules
        |
        v
V8 worker output
        |
        +--> local run with workerd
        +--> Cloudflare package in dist/cloudflare/
        +--> hosted Distlang deploy
```

## How It Works Here

The current repo is a Phase 0 POC with a JavaScript worker build path and Cloudflare-first packaging.

- Plain worker apps can import `distlang/core` to get storage helpers like `InMemDB`, which use memory locally and Cloudflare KV when available.
- Higher-level apps can use `distlang/app` to define routes, state, DBs, and observability in a more structured app shape.
- `build` writes generated output into the app directory, including `dist/` and `generated/`.
- `run` builds the V8 backend and launches it locally through `workerd`.
- `deploy` targets Distlang hosting by default, or direct Cloudflare deployment with `--target=cloudflare`.

## Commands
- `build <file>`: build backend artifacts (`v8`) and provider packaging into `dist/`.
- `target init [--target=cloudflare] [--path=.]`: scaffold target files (including local env template) for a project/example.
- `deploy <file> [--target=distlang|cloudflare]`: deploy through Distlang hosting by default, or directly to Cloudflare with `--target=cloudflare`.
- `helpers <login|store|whoami|logout>`: manage the Distlang helper auth session and authenticated store access.
- `run <file> [--v8-port=N]`: build the V8 backend, then start local workerd.
- `debug <build|run> <file> [--passes=...]`: print pass outputs (`parse`, `ir`, `emit`); `debug run` now points you to `distlang run`.

## Build And Deploy Apps

Build artifacts include backend output in `dist/` and any generated distlang helper code in `generated/`.

Hosted deploys currently:
- infer the app name from the parent directory
- support single-worker apps first
- publish into Distlang-managed Cloudflare hosting
- return a hosted URL like `https://echo-georg-a1b2c3.distlang.com`

Deploy directly to your own Cloudflare account:
```bash
distlang deploy path/to/index.js --target=cloudflare
```

That direct-provider path still expects Cloudflare credentials and target config.

## Releases and CI Artifacts

### Release assets (public + stable)
Tagging a release (for example `v0.1.0`) triggers `.github/workflows/release.yml` and publishes cross-platform binaries to GitHub Releases.

### Rolling `main` assets (public + prerelease)
Pushes to `main` trigger `.github/workflows/main-release.yml` and update a rolling prerelease named `main`.

This powers:
```bash
curl -fsSL https://distlang.com/install-main | bash
```

Main-channel binaries identify themselves clearly, for example:
```bash
distlang main (985e910)
```

Download the latest Linux AMD64 build:
```bash
curl -fsSL -o distlang_linux_amd64.tar.gz \
  https://github.com/distlanglabs/distlang/releases/latest/download/distlang_linux_amd64.tar.gz
tar -xzf distlang_linux_amd64.tar.gz
chmod +x distlang_linux_amd64
./distlang_linux_amd64 --help
```

Download a pinned version:
```bash
VERSION=v0.1.0
curl -fsSL -o distlang_darwin_arm64.tar.gz \
  "https://github.com/distlanglabs/distlang/releases/download/${VERSION}/distlang_darwin_arm64.tar.gz"
```

Windows release assets are published as `.zip` files (for example `distlang_windows_amd64.zip`).

Verify checksums:
```bash
curl -fsSL -o checksums.txt \
  https://github.com/distlanglabs/distlang/releases/latest/download/checksums.txt
grep 'distlang_linux_amd64.tar.gz' checksums.txt | sha256sum -c -
```

### CI workflow artifacts (short-lived)
Every run of `.github/workflows/ci.yml` uploads a Linux AMD64 CLI artifact per Go version. These artifacts are for CI/debug usage and expire automatically.

From the GitHub UI:
1. Open the workflow run.
2. Download from the **Artifacts** section at the bottom.

From GitHub CLI:
```bash
# list recent runs
gh run list --workflow ci.yml

# download artifacts from a specific run id
gh run download <run-id>
```

### Maintainer release flow
```bash
# 1) create and push a tag
git tag v0.1.0
git push origin v0.1.0

# 2) release workflow builds and publishes assets
#    - distlang_<os>_<arch>.tar.gz
#    - checksums.txt
```

Local release asset build commands:
```bash
make build-cross
make package
make checksums
```

## Helper Auth
- `distlang helpers login`: opens the browser for Google auth and stores a local session.
- `distlang helpers auth status [--json]`: reports whether a helper auth session is available.
- `distlang helpers request <METHOD> <PATH_OR_URL> [--body-file=...] [--content-type=...] [--json]`: makes an authenticated public API request using the saved helper session.
- `distlang helpers store objectdb status`: confirms store access for the logged-in user.
- `distlang helpers store objectdb buckets list|create|exists|delete`: manage ObjectDB buckets.
- `distlang helpers store objectdb keys list <bucket>`: list keys in a bucket with optional pagination flags.
- `distlang helpers store objectdb put|get|head|delete <bucket> <key>`: manage ObjectDB values; `keys` only supports `list`.
- `distlang helpers whoami`: refreshes the session when needed and prints the current user.
- `distlang helpers logout`: revokes the remote refresh token and clears the local session.
- `DISTLANG_AUTH_BASE_URL`: optional auth service override for local testing or staging; defaults to `https://auth.distlang.com`.
- `DISTLANG_STORE_BASE_URL`: optional store service override for local testing or staging; defaults to `https://api.distlang.com`.

### Staging helpers

Use these overrides to talk to staging instead of production:

```bash
export DISTLANG_AUTH_BASE_URL=https://auth-staging.distlang.com
export DISTLANG_STORE_BASE_URL=https://api-staging.distlang.com
distlang helpers login
```

Helper sessions are stored per auth host, so staging login does not overwrite the production session.

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
cd examples/app-echo
make build
# outputs summary and writes dist/ in the example directory
```

Deploy the example through Distlang hosting:
```bash
distlang deploy examples/app-echo/index.js
```

Deploy directly to Cloudflare instead:
```bash
distlang deploy examples/app-echo/index.js --target=cloudflare
```

## Known Limitations
- `run` requires external `workerd`.
- No hot reload; worker processes are launched once per run invocation.
- Strict worker-style entrypoints are still assumed on the V8 path.

## Roadmap
See [ROADMAP.md](ROADMAP.md) for vision, architecture, and upcoming phases.
