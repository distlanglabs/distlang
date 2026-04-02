# app-echo example

This example shows the new `app(...)` shape with explicit `state` and `compute` sections.

## Shape

```js
app({
  state: {
    dbs: {
      ObjectDB: helpers.ObjectDB,
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

## Build

From this directory:

```bash
../../bin/distlang build index.js
```
