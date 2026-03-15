# helloworld example

Run commands from this directory.

```bash
# show available commands
make help

# create local Cloudflare target files once
make cloudflare-init
# then set values in targets/cloudflare/cloudflare.env

# run the local v8 runtime
make run

# inspect compile/debug passes
make debug

# build artifacts
make build

# inspect generated distlang helpers
ls generated/distlang/core

# Cloudflare helper commands route through generated dist/cloudflare/Makefile
make cloudflare-deps  # install/check wrangler
make dev              # wrangler dev

# deploy to cloudflare (same route, but wrapped)
make deploy
```

`make` targets automatically build the distlang CLI in the repo root first.
`make run` expects local `workerd`.
`make deploy` builds the example, loads `targets/cloudflare/cloudflare.env`, and then runs `make -C dist/cloudflare publish`.
`make build` also writes visible generated helper code under `generated/distlang/core/index.js` when the example imports `distlang/core` and uses `InMemDB`.
