# helloworld example

Run commands from this directory.

```bash
# show available commands
make help

# create local Cloudflare target files once
make target-init
# then set values in targets/cloudflare/cloudflare.env

# run worker locally
make run

# inspect compile/debug passes
make debug

# build artifacts
make build

# deploy to cloudflare
make deploy
```

`make` targets automatically build the distlang CLI in the repo root first.
