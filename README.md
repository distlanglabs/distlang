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

Build a binary:

```bash
make build
./bin/distlang
```
