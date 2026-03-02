# distlang

# Overview

distlang dev <folder> # compiles, brings up the port locally running the project. 

jsProject which understand winterTC -> wasm -> run on a port.

## Overview diagram

```text
                           +----------------------+
                           |      distlang CLI    |
                           |   (Go orchestrator)  |
                           +----------+-----------+
                                      |
          +---------------------------+----------------------------+
          |                            |                           |
          v                            v                           v
+---------------------+     +----------------------+    +----------------------+
|  Compiler/Planner   |     |   Deploy/Control     |    | Version/Lock Manager |
|      (Go)           |     |      Plane (Go)      |    |        (Go)          |
+----------+----------+     +----------+-----------+    +----------+-----------+
           |                           |                           |
           | produces                  | consumes                  | tracks
           v                           v                           v
                 +-----------------------------------------------+
                 |         Distlang IR + Capability ABI          |
                 |     (versioned JSON contract, stable)         |
                 +-------------------+---------------------------+
                                     |
                                     | invokes runtime backends
                                     v
     +-------------------+--------------------+--------------------+-------------------+
     |                   |                    |                    |                   |
     v                   v                    v                    v                   v
+-----------+     +-------------+      +-------------+      +-------------+     +-------------+
| Rust Comp |     | Node Runner |      | workerd Run |      | Deno Runner |     | Future WASI |
| (Wasm/CM) |     |  (subproc)  |      |  (subproc)  |      |  (subproc)  |     |   backend   |
+-----+-----+     +------+------+      +------+------+      +------+------+     +------+
      |                  |                    |                    |                   |
      +------------------+--------------------+--------------------+-------------------+
                                     |
                                     v
                        +-------------------------------+
                        | Vendor Adapters / Providers   |
                        | Cloudflare | AWS | Azure | Local |
                        +-------------------------------+
```

# This is Cloudflare worker code. 

```
export default {
  async fetch(request, env, ctx) {
    return new Response("Hello Worker!");
  },
};
```

distlang should convert this to wasm and run it locally. 



# POC

## Phase 0: Local POC

- Node-based local runtime
- WinterTC-style `Request`/`Response`
- Capabilities: `http`, `kv`, `log`
- Echo route + in-memory KV

## Run locally

```bash
make run
```

Show help text:

```bash
go run ./cmd/distlang -h
```

Full help (global + per-command):

```bash
./bin/distlang --full-help
```

Build a binary:

```bash
make build
./bin/distlang
```

Run the POC build command:

```bash
./bin/distlang build examples/helloworld/index.js
```

`build` now runs the compile pipeline and prints the Goja-ready JS output.

Run JS (ESM is transformed to Goja-friendly script first):

```bash
./bin/distlang run examples/helloworld/index.js
./bin/distlang run examples/helloworld/console.js
```

Debug compiler passes (parse, ir, emit):

```bash
./bin/distlang debug build examples/helloworld/index.js --passes=parse,ir,emit
./bin/distlang debug run examples/helloworld/console.js --passes=ir,emit
./bin/distlang debug --help
```
