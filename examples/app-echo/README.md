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

It keeps metrics scalar by default so dashboards stay readable. Add labels only for small, bounded dimensions when you need a split view.

## Build

From this directory:

```bash
../../bin/distlang build index.js
```
