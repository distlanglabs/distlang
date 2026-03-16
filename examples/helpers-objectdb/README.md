# helpers-objectdb example

Run commands from this directory.

```bash
# show available commands
make help

# create local Cloudflare target files once
make cloudflare-init

# run local v8 runtime
make run

# build artifacts
make build
```

Example routes:

```bash
# status
curl "http://127.0.0.1:5656/"

# create bucket
curl -X PUT "http://127.0.0.1:5656/bucket/demo"

# write value
curl -X PUT "http://127.0.0.1:5656/bucket/demo/key/hello" \
  -H "Content-Type: application/json" \
  -d '{"message":"hi"}'

# read value
curl "http://127.0.0.1:5656/bucket/demo/key/hello"

# delete value
curl -X DELETE "http://127.0.0.1:5656/bucket/demo/key/hello"
```

Local runtime defaults to mock helpers mode unless `DISTLANG_SERVICE_TOKEN` is provided.
For local live-mode testing, set:

- `DISTLANG_HELPERS_MODE=live`
- `DISTLANG_STORE_BASE_URL=http://127.0.0.1:<port>`
- `DISTLANG_SERVICE_TOKEN=<token>`

During deploy, if this example imports `distlang`, distlang will request a service token from auth for the logged-in user and inject it as Cloudflare secret `DISTLANG_SERVICE_TOKEN`.

Run local smoke coverage:

```bash
node ../../test/runtime/helpers_objectdb_smoke.mjs
```
