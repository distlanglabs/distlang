import http from "node:http";

function json(res, status, payload, extraHeaders = {}) {
  const body = JSON.stringify(payload);
  res.writeHead(status, {
    "Content-Type": "application/json",
    "Content-Length": Buffer.byteLength(body),
    ...extraHeaders,
  });
  res.end(body);
}

function notFound(res) {
  json(res, 404, { error: "not_found", message: "not found" });
}

function parseBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => resolve(Buffer.concat(chunks)));
    req.on("error", reject);
  });
}

function decodeValue(raw, contentType) {
  if (contentType.includes("application/json")) {
    if (raw.length === 0) {
      return null;
    }
    return JSON.parse(raw.toString("utf8"));
  }
  return raw.toString("utf8");
}

export async function startMockHelpersServer(options = {}) {
  const token = String(options.token || "local-service-token");
  const buckets = new Map();
  const analyticsBuckets = new Set();
  const analyticsRows = new Map();
  const metricsBuckets = new Set();
  const metricsMetadata = new Map();
  const metricsRows = new Map();
  const calls = [];

  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, "http://127.0.0.1");
    calls.push({ method: req.method, path: url.pathname, search: url.search });

    const auth = req.headers.authorization || "";
    if (auth !== `Bearer ${token}`) {
      json(res, 401, { error: "unauthorized", message: "missing or invalid bearer token" });
      return;
    }

    if (req.method === "GET" && url.pathname === "/objectdb/v1") {
      json(res, 200, {
        ok: true,
        service: "objectdb",
        version: "mock",
        routes: {
          buckets: "/objectdb/v1/buckets",
          keys: "/objectdb/v1/buckets/:bucket/keys",
          values: "/objectdb/v1/buckets/:bucket/values/:key",
        },
      });
      return;
    }

    if (req.method === "GET" && url.pathname === "/metrics/v1") {
      json(res, 200, {
        ok: true,
        service: "metrics",
        version: "mock",
        routes: {
          buckets: "/metrics/v1/buckets/:bucket",
          metadata: "/metrics/v1/buckets/:bucket/metadata",
          rows: "/metrics/v1/buckets/:bucket/rows",
          query: "/metrics/v1/api/v1/query",
          query_range: "/metrics/v1/api/v1/query_range",
        },
      });
      return;
    }

    if (req.method === "GET" && url.pathname === "/metrics/v1/api/v1/query") {
      const query = url.searchParams.get("query") || "";
      const time = url.searchParams.get("time");
      json(res, 200, mockMetricsQuery(metricsRows, query, { time }));
      return;
    }

    if (req.method === "GET" && url.pathname === "/metrics/v1/api/v1/query_range") {
      const query = url.searchParams.get("query") || "";
      const start = url.searchParams.get("start");
      const end = url.searchParams.get("end");
      const step = url.searchParams.get("step");
      json(res, 200, mockMetricsQueryRange(metricsRows, query, { start, end, step }));
      return;
    }

    if (req.method === "GET" && url.pathname === "/metrics/v1/api/v1/series") {
      const match = url.searchParams.getAll("match[]");
      json(res, 200, mockMetricsSeries(metricsRows, { match }));
      return;
    }

    if (req.method === "GET" && url.pathname === "/metrics/v1/api/v1/labels") {
      const match = url.searchParams.getAll("match[]");
      json(res, 200, mockMetricsLabels(metricsRows, { match }));
      return;
    }

    const labelValuesMatch = url.pathname.match(/^\/metrics\/v1\/api\/v1\/label\/([^/]+)\/values$/);
    if (labelValuesMatch && req.method === "GET") {
      const match = url.searchParams.getAll("match[]");
      json(res, 200, mockMetricsLabelValues(metricsRows, decodeURIComponent(labelValuesMatch[1]), { match }));
      return;
    }

    if (req.method === "GET" && url.pathname === "/metrics/v1/api/v1/metadata") {
      const metric = url.searchParams.get("metric") || "";
      json(res, 200, mockMetricsMetadata(metricsMetadata, { metric }));
      return;
    }

    if (url.pathname === "/objectdb/v1/buckets" && req.method === "GET") {
      const list = Array.from(buckets.keys()).sort().map((name) => ({ name, createdAt: new Date().toISOString() }));
      json(res, 200, { ok: true, buckets: list });
      return;
    }

    const bucketMatch = url.pathname.match(/^\/objectdb\/v1\/buckets\/([^/]+)$/);
    if (bucketMatch) {
      const bucket = decodeURIComponent(bucketMatch[1]);
      if (req.method === "PUT") {
        const created = !buckets.has(bucket);
        if (!buckets.has(bucket)) {
          buckets.set(bucket, new Map());
        }
        json(res, 200, { ok: true, bucket, created });
        return;
      }
      if (req.method === "DELETE") {
        const deleted = buckets.delete(bucket);
        json(res, 200, { ok: true, bucket, deleted });
        return;
      }
      notFound(res);
      return;
    }

    const keysMatch = url.pathname.match(/^\/objectdb\/v1\/buckets\/([^/]+)\/keys$/);
    if (keysMatch && req.method === "GET") {
      const bucket = decodeURIComponent(keysMatch[1]);
      const records = buckets.get(bucket);
      if (!records) {
        json(res, 200, { ok: true, bucket, keys: [], list_complete: true, cursor: "" });
        return;
      }
      const prefix = url.searchParams.get("prefix") || "";
      const limit = Number.parseInt(url.searchParams.get("limit") || "1000", 10);
      const all = Array.from(records.entries())
        .filter(([key]) => key.startsWith(prefix))
        .map(([key, record]) => ({
          name: key,
          expiration: null,
          metadata: {
            contentType: record.contentType,
            size: record.raw.length,
            updatedAt: record.updatedAt,
          },
        }));
      const keys = all.slice(0, Number.isFinite(limit) && limit > 0 ? limit : 1000);
      json(res, 200, { ok: true, bucket, keys, list_complete: keys.length === all.length, cursor: "" });
      return;
    }

    const metricsBucketMatch = url.pathname.match(/^\/metrics\/v1\/buckets\/([^/]+)$/);
    if (metricsBucketMatch && req.method === "PUT") {
      const bucket = decodeURIComponent(metricsBucketMatch[1]);
      const created = !metricsBuckets.has(bucket);
      metricsBuckets.add(bucket);
      if (!metricsRows.has(bucket)) {
        metricsRows.set(bucket, []);
      }
      json(res, 200, { ok: true, bucket, created });
      return;
    }

    const metricsMetadataMatch = url.pathname.match(/^\/metrics\/v1\/buckets\/([^/]+)\/metadata$/);
    if (metricsMetadataMatch && req.method === "PUT") {
      const bucket = decodeURIComponent(metricsMetadataMatch[1]);
      const raw = await parseBody(req);
      const body = raw.length === 0 ? {} : JSON.parse(raw.toString("utf8"));
      metricsMetadata.set(bucket, body.metrics || {});
      json(res, 200, { ok: true, bucket, metrics: body.metrics || {} });
      return;
    }

    const metricsRowsMatch = url.pathname.match(/^\/metrics\/v1\/buckets\/([^/]+)\/rows$/);
    if (metricsRowsMatch && req.method === "POST") {
      const bucket = decodeURIComponent(metricsRowsMatch[1]);
      if (!metricsBuckets.has(bucket)) {
        metricsBuckets.add(bucket);
      }
      if (!metricsRows.has(bucket)) {
        metricsRows.set(bucket, []);
      }
      const raw = await parseBody(req);
      const body = raw.length === 0 ? {} : JSON.parse(raw.toString("utf8"));
      const rows = Array.isArray(body.rows) ? body.rows : [];
      metricsRows.get(bucket).push(...rows);
      json(res, 201, { ok: true, bucket, written: rows.length });
      return;
    }

    const valueMatch = url.pathname.match(/^\/objectdb\/v1\/buckets\/([^/]+)\/values\/(.+)$/);
    if (valueMatch) {
      const bucket = decodeURIComponent(valueMatch[1]);
      const key = decodeURIComponent(valueMatch[2]);
      const records = buckets.get(bucket);
      if (req.method === "PUT") {
        if (!records) {
          json(res, 404, { error: "bucket_not_found", message: "bucket does not exist" });
          return;
        }
        const raw = await parseBody(req);
        const contentType = String(req.headers["content-type"] || "application/octet-stream");
        const value = decodeValue(raw, contentType);
        const updatedAt = new Date().toISOString();
        records.set(key, { raw, value, contentType, updatedAt });
        json(res, 200, {
          ok: true,
          bucket,
          key,
          metadata: {
            contentType,
            size: raw.length,
            updatedAt,
          },
        });
        return;
      }
      if (req.method === "GET") {
        if (!records || !records.has(key)) {
          json(res, 404, { error: "key_not_found", message: "No value exists for that key." });
          return;
        }
        const record = records.get(key);
        const type = url.searchParams.get("type") || "json";
        if (type === "json") {
          json(res, 200, record.value, {
            "X-Distlang-Value-Size": String(record.raw.length),
            "X-Distlang-Updated-At": record.updatedAt,
          });
        } else {
          res.writeHead(200, {
            "Content-Type": record.contentType,
            "X-Distlang-Value-Size": String(record.raw.length),
            "X-Distlang-Updated-At": record.updatedAt,
          });
          res.end(record.raw);
        }
        return;
      }
      if (req.method === "DELETE") {
        const deleted = !!(records && records.delete(key));
        json(res, 200, { ok: true, bucket, key, deleted });
        return;
      }
      notFound(res);
      return;
    }

    notFound(res);
  });

  await new Promise((resolve, reject) => {
    server.listen(0, "127.0.0.1", (err) => {
      if (err) {
        reject(err);
        return;
      }
      resolve();
    });
  });

  const address = server.address();
  const port = typeof address === "object" && address ? address.port : 0;

  return {
    token,
    calls,
    analyticsRows: metricsRows,
    baseURL: `http://127.0.0.1:${port}`,
    close: async () => {
      await new Promise((resolve, reject) => {
        server.close((err) => {
          if (err) {
            reject(err);
            return;
          }
          resolve();
        });
      });
    },
  };
}

function allMetricRows(store) {
  const rows = [];
  for (const [bucket, bucketRows] of store.entries()) {
    for (const row of bucketRows) {
      rows.push({ bucket, row });
    }
  }
  return rows;
}

function mockMetricsQuery(store, query, options = {}) {
  const parsed = parseQuery(query);
  const evalTime = parseTime(options.time) || Date.now();
  const grouped = groupSeries(store, parsed.selector);
  const result = [];
  for (const entry of grouped.values()) {
    const value = evalSeries(parsed, entry.points, evalTime);
    if (value == null) {
      continue;
    }
    result.push({ metric: entry.metric, value: [evalTime / 1000, String(value)] });
  }
  return { status: "success", data: { resultType: "vector", result } };
}

function mockMetricsQueryRange(store, query, options = {}) {
  const parsed = parseQuery(query);
  const start = parseTime(options.start);
  const end = parseTime(options.end);
  const step = parseStep(options.step);
  const matrix = new Map();
  for (let ts = start; ts <= end; ts += step) {
    const vector = mockMetricsQuery(store, query, { time: String(ts / 1000) }).data.result;
    for (const item of vector) {
      const key = JSON.stringify(item.metric);
      const entry = matrix.get(key) || { metric: item.metric, values: [] };
      entry.values.push(item.value);
      matrix.set(key, entry);
    }
  }
  return { status: "success", data: { resultType: "matrix", result: Array.from(matrix.values()) } };
}

function mockMetricsSeries(store, options = {}) {
  const selectors = Array.isArray(options.match) ? options.match.map(parseSelector) : [];
  const series = new Map();
  for (const { bucket, row } of allMetricRows(store)) {
    const labels = rowLabels(bucket, row.data || row);
    if (selectors.length > 0 && !selectors.some((selector) => matchesSelector(labels, selector))) {
      continue;
    }
    series.set(JSON.stringify(labels), labels);
  }
  return { status: "success", data: Array.from(series.values()) };
}

function mockMetricsLabels(store, options = {}) {
  const labels = new Set();
  for (const item of mockMetricsSeries(store, options).data) {
    for (const key of Object.keys(item)) {
      labels.add(key);
    }
  }
  return { status: "success", data: Array.from(labels).sort() };
}

function mockMetricsLabelValues(store, name, options = {}) {
  const values = new Set();
  for (const item of mockMetricsSeries(store, options).data) {
    if (typeof item[name] === "string") {
      values.add(item[name]);
    }
  }
  return { status: "success", data: Array.from(values).sort() };
}

function mockMetricsMetadata(store, options = {}) {
  const data = {};
  for (const [bucket, definitions] of store.entries()) {
    for (const [name, definition] of Object.entries(definitions)) {
      if (options.metric && options.metric !== name) {
        continue;
      }
      if (!Array.isArray(data[name])) {
        data[name] = [];
      }
      data[name].push({ type: definition.kind, help: definition.description, unit: definition.unit, bucket, labels: (definition.labels || []).join(",") });
    }
  }
  return { status: "success", data };
}

function parseQuery(rawQuery) {
  const query = String(rawQuery || "").trim();
  const fnMatch = query.match(/^([a-zA-Z_][a-zA-Z0-9_]*)\((.+)\)$/);
  if (fnMatch) {
    const rangeMatch = fnMatch[2].trim().match(/^(.*)\[([^\]]+)\]$/);
    return { kind: "rangeFunction", fn: fnMatch[1], selector: parseSelector(rangeMatch[1].trim()), rangeMs: parseDuration(rangeMatch[2].trim()) };
  }
  return { kind: "selector", selector: parseSelector(query) };
}

function parseSelector(rawSelector) {
  const match = String(rawSelector || "").trim().match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{(.*)\})?$/);
  const matchers = { __name__: match[1] };
  if (match[2]) {
    for (const part of match[2].split(",")) {
      const matcher = part.trim().match(/^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"([^"]*)"$/);
      matchers[matcher[1]] = matcher[2];
    }
  }
  return { metric: match[1], matchers };
}

function groupSeries(store, selector) {
  const grouped = new Map();
  for (const { bucket, row } of allMetricRows(store)) {
    const data = row.data || row;
    const labels = rowLabels(bucket, data);
    if (!matchesSelector(labels, selector)) {
      continue;
    }
    const key = JSON.stringify(labels);
    const entry = grouped.get(key) || { metric: labels, points: [] };
    if (data.kind === "histogram") {
      for (const value of data.values || []) {
        entry.points.push({ ts: Date.parse(data.windowStart), value });
      }
    } else {
      entry.points.push({ ts: Date.parse(data.windowStart), value: data.sum });
    }
    grouped.set(key, entry);
  }
  return grouped;
}

function rowLabels(bucket, data) {
  return { __name__: data.metric, bucket, ...(data.labels || {}) };
}

function matchesSelector(labels, selector) {
  for (const [key, value] of Object.entries(selector.matchers)) {
    if (labels[key] !== value) {
      return false;
    }
  }
  return true;
}

function evalSeries(parsed, points, evalTime) {
  const lookback = 5 * 60 * 1000;
  if (parsed.kind === "selector") {
    const latest = points.filter((point) => point.ts <= evalTime && point.ts >= evalTime - lookback).sort((a, b) => a.ts - b.ts).pop();
    return latest ? latest.value : null;
  }
  const values = points.filter((point) => point.ts >= evalTime - parsed.rangeMs && point.ts <= evalTime).map((point) => point.value);
  if (values.length === 0) {
    return null;
  }
  switch (parsed.fn) {
    case "sum_over_time":
    case "increase":
      return values.reduce((sum, value) => sum + value, 0);
    case "rate":
      return values.reduce((sum, value) => sum + value, 0) / (parsed.rangeMs / 1000);
    case "avg_over_time":
      return values.reduce((sum, value) => sum + value, 0) / values.length;
    case "p50":
      return percentile(values, 50);
    case "p90":
      return percentile(values, 90);
    case "p95":
      return percentile(values, 95);
    case "p99":
      return percentile(values, 99);
    default:
      return null;
  }
}

function percentile(values, pct) {
  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil((pct / 100) * sorted.length) - 1));
  return sorted[index];
}

function parseTime(value) {
  const raw = String(value || "").trim();
  const numeric = Number(raw);
  if (Number.isFinite(numeric)) {
    return numeric * 1000;
  }
  return Date.parse(raw);
}

function parseStep(value) {
  const raw = String(value || "").trim();
  const numeric = Number(raw);
  if (Number.isFinite(numeric)) {
    return numeric * 1000;
  }
  return parseDuration(raw);
}

function parseDuration(raw) {
  const units = { ms: 1, s: 1000, m: 60 * 1000, h: 60 * 60 * 1000, d: 24 * 60 * 60 * 1000 };
  let total = 0;
  let consumed = 0;
  let match;
  const pattern = /(\d+(?:\.\d+)?)(ms|s|m|h|d)/g;
  while ((match = pattern.exec(raw)) !== null) {
    total += Number(match[1]) * units[match[2]];
    consumed += match[0].length;
  }
  return consumed === raw.length ? total : 0;
}
