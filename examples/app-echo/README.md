# app-echo example

This example shows the new `app(...)` shape with explicit `state` and `compute` sections.

## Shape

```js
app({
  state: {
    dbs: {
      ObjectDB: helpers.ObjectDB,
    },
    observability: {
      AppMetrics: helpers.instantiateMetrics(...),
    },
  },
  compute: {
    handlers: appHandlers,
  },
})
```

## Routes

- `POST /echo/config`
- `GET /echo/:text`

The example records request metrics through `state.observability.AppMetrics` while storing config in `state.dbs.ObjectDB`.

## Build

From this directory:

```bash
../../bin/distlang build index.js
```
