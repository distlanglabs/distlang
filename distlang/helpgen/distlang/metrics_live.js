import { encodePathPart, queryOptions, requestJSON } from "./shared.js";

export async function liveMetricsQuery(cfg, query, options = {}) {
  return requestJSON("GET", "/metrics/v1/api/v1/query", cfg, {
    query: queryOptions({ query, time: options.time }),
    errorPrefix: "helpers.Metrics request",
  });
}

export async function liveMetricsQueryRange(cfg, query, options = {}) {
  return requestJSON("GET", "/metrics/v1/api/v1/query_range", cfg, {
    query: queryOptions({ query, start: options.start, end: options.end, step: options.step }),
    errorPrefix: "helpers.Metrics request",
  });
}

export async function liveMetricsSeries(cfg, options = {}) {
  return requestJSON("GET", "/metrics/v1/api/v1/series", cfg, {
    query: queryOptions({ "match[]": options.match, start: options.start, end: options.end }),
    errorPrefix: "helpers.Metrics request",
  });
}

export async function liveMetricsLabels(cfg, options = {}) {
  return requestJSON("GET", "/metrics/v1/api/v1/labels", cfg, {
    query: queryOptions({ "match[]": options.match, start: options.start, end: options.end }),
    errorPrefix: "helpers.Metrics request",
  });
}

export async function liveMetricsLabelValues(cfg, name, options = {}) {
  return requestJSON("GET", `/metrics/v1/api/v1/label/${encodePathPart(name)}/values`, cfg, {
    query: queryOptions({ "match[]": options.match, start: options.start, end: options.end }),
    errorPrefix: "helpers.Metrics request",
  });
}

export async function liveMetricsMetadata(cfg, options = {}) {
  return requestJSON("GET", "/metrics/v1/api/v1/metadata", cfg, {
    query: queryOptions({ metric: options.metric }),
    errorPrefix: "helpers.Metrics request",
  });
}

export async function ensureLiveMetricsBucket(cfg, bucket, definitions) {
  await requestJSON("PUT", `/metrics/v1/buckets/${encodePathPart(bucket)}`, cfg, {
    errorPrefix: "helpers.Metrics request",
  });
  await requestJSON("PUT", `/metrics/v1/buckets/${encodePathPart(bucket)}/metadata`, cfg, {
    body: JSON.stringify({ metrics: definitions }),
    headers: { "Content-Type": "application/json" },
    errorPrefix: "helpers.Metrics request",
  });
}

export async function appendLiveMetricRows(cfg, bucket, rows) {
  await requestJSON("POST", `/metrics/v1/buckets/${encodePathPart(bucket)}/rows`, cfg, {
    body: JSON.stringify({ rows: rows.map((row) => ({ ts: row.windowStart, data: row })) }),
    headers: { "Content-Type": "application/json" },
    errorPrefix: "helpers.Metrics request",
  });
}
