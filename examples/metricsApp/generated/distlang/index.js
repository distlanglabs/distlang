import { InMemDB } from "distlang/core";

const defaultStoreBaseURL = "https://api.distlang.com";
let currentEnv = null;
let currentCtx = null;
const mockBuckets = new Set();
const mockMetricsBuckets = new Set();
const mockMetricsRows = new Map();
const mockMetricsMetadata = new Map();
const metricsStates = new Set();
const metricsWindowMs = 30 * 1000;

export function wrapWorkerWithHelpers(worker) {
  return {
    ...worker,
    async fetch(request, env, ctx) {
      currentEnv = env || null;
      currentCtx = ctx || null;
      try {
        return await worker.fetch(request, env, ctx);
      } finally {
        queueFlushAllMetricsStates();
        if (!currentCtx || typeof currentCtx.waitUntil !== "function") {
          await settleAllMetricFlushes();
        }
        currentCtx = null;
      }
    },
  };
}

function envString(key) {
  if (!currentEnv) {
    return "";
  }
  const value = currentEnv[key];
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}

function helpersMode() {
  const mode = envString("DISTLANG_HELPERS_MODE").toLowerCase();
  if (mode === "mock" || mode === "live") {
    return mode;
  }
  return "auto";
}

function liveConfig(errorPrefix) {
  const mode = helpersMode();
  const token = envString("DISTLANG_SERVICE_TOKEN");
  const baseURL = (envString("DISTLANG_STORE_BASE_URL") || defaultStoreBaseURL).replace(/\/$/, "");
  const shouldUseLive = mode === "live" || (mode === "auto" && token !== "");

  if (!shouldUseLive) {
    return { live: false };
  }
  if (token === "") {
    throw new Error(`${errorPrefix} requires DISTLANG_SERVICE_TOKEN in live mode`);
  }

  return { live: true, token, baseURL };
}

function bucketKey(bucket, key) {
  return `${bucket}:${key}`;
}

function listOptions(options = {}) {
  const out = [];
  if (typeof options.prefix === "string" && options.prefix !== "") {
    out.push(["prefix", options.prefix]);
  }
  if (typeof options.limit === "number" && Number.isFinite(options.limit) && options.limit > 0) {
    out.push(["limit", String(Math.floor(options.limit))]);
  }
  if (typeof options.cursor === "string" && options.cursor !== "") {
    out.push(["cursor", options.cursor]);
  }
  return out;
}

function encodePathPart(value) {
  return encodeURIComponent(String(value));
}

function queryOptions(options = {}) {
  const out = [];
  for (const [key, value] of Object.entries(options)) {
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item != null && String(item) !== "") {
          out.push([key, String(item)]);
        }
      }
      continue;
    }
    if (value != null && String(value) !== "") {
      out.push([key, String(value)]);
    }
  }
  return out;
}

async function requestJSON(method, path, cfg, options = {}) {
  const url = new URL(path, cfg.baseURL);
  if (Array.isArray(options.query)) {
    for (const [key, value] of options.query) {
      url.searchParams.set(key, value);
    }
  }

  const headers = {
    Authorization: `Bearer ${cfg.token}`,
    ...options.headers,
  };
  const res = await fetch(url.toString(), {
    method,
    headers,
    body: options.body,
  });

  if (res.status === 404 && options.allowNotFound) {
    return null;
  }

  const text = await res.text();
  let payload = null;
  if (text !== "") {
    try {
      payload = JSON.parse(text);
    } catch (_err) {
      payload = text;
    }
  }

  if (!res.ok) {
    const message = payload && typeof payload === "object" && payload.message ? payload.message : text || `${res.status} ${res.statusText}`;
    const prefix = typeof options.errorPrefix === "string" && options.errorPrefix !== ""
      ? options.errorPrefix
      : "helpers request";
    throw new Error(`${prefix} failed (${res.status}): ${message}`);
  }

  if (options.expectText) {
    return text;
  }
  return payload;
}

async function mockStatus() {
  return {
    ok: true,
    service: "objectdb",
    mode: "mock",
  };
}

async function mockBucketsList() {
  return {
    ok: true,
    buckets: Array.from(mockBuckets).sort().map((name) => ({ name })),
  };
}

async function mockBucketsCreate(bucket) {
  const name = String(bucket);
  const created = !mockBuckets.has(name);
  mockBuckets.add(name);
  return {
    ok: true,
    bucket: name,
    created,
  };
}

async function mockBucketsExists(bucket) {
  return mockBuckets.has(String(bucket));
}

async function mockBucketsDelete(bucket) {
  const name = String(bucket);
  const deleted = mockBuckets.delete(name);
  const keys = await InMemDB.list({ prefix: `${name}:` });
  await Promise.all(keys.map((key) => InMemDB.delete(key)));
  return {
    ok: true,
    bucket: name,
    deleted,
  };
}

async function mockKeysList(bucket, options = {}) {
  const name = String(bucket);
  if (!mockBuckets.has(name)) {
    return {
      ok: true,
      bucket: name,
      keys: [],
      list_complete: true,
      cursor: "",
    };
  }

  const prefix = typeof options.prefix === "string" ? options.prefix : "";
  const fullPrefix = `${name}:${prefix}`;
  const all = await InMemDB.list({ prefix: fullPrefix });
  const keys = all.map((value) => value.slice(name.length + 1));
  const limit = typeof options.limit === "number" && options.limit > 0 ? Math.floor(options.limit) : 1000;
  return {
    ok: true,
    bucket: name,
    keys: keys.slice(0, limit).map((key) => ({ name: key, expiration: null, metadata: null })),
    list_complete: keys.length <= limit,
    cursor: "",
  };
}

async function mockPut(bucket, key, value) {
  const bucketName = String(bucket);
  if (!mockBuckets.has(bucketName)) {
    throw new Error(`helpers.ObjectDB.put: bucket does not exist: ${bucketName}`);
  }
  await InMemDB.put(bucketKey(bucketName, key), value);
  return {
    ok: true,
    bucket: bucketName,
    key: String(key),
  };
}

async function mockGet(bucket, key) {
  const bucketName = String(bucket);
  if (!mockBuckets.has(bucketName)) {
    return null;
  }
  return InMemDB.get(bucketKey(bucketName, key));
}

async function mockHead(bucket, key) {
  const bucketName = String(bucket);
  const value = await mockGet(bucketName, key);
  if (value == null) {
    return null;
  }
  return {
    content_type: "application/json",
    content_size: JSON.stringify(value).length,
  };
}

async function mockDelete(bucket, key) {
  const bucketName = String(bucket);
  if (!mockBuckets.has(bucketName)) {
    return {
      ok: true,
      bucket: bucketName,
      key: String(key),
      deleted: false,
    };
  }
  const deleted = await InMemDB.delete(bucketKey(bucketName, key));
  return {
    ok: true,
    bucket: bucketName,
    key: String(key),
    deleted,
  };
}

const helpersObjectDB = {
  async status() {
    const cfg = liveConfig("helpers.ObjectDB");
    if (!cfg.live) {
      return mockStatus();
    }
    return requestJSON("GET", "/objectdb/v1", cfg, { errorPrefix: "helpers.ObjectDB request" });
  },

  buckets: {
    async list() {
      const cfg = liveConfig("helpers.ObjectDB");
      if (!cfg.live) {
        return mockBucketsList();
      }
      return requestJSON("GET", "/objectdb/v1/buckets", cfg, { errorPrefix: "helpers.ObjectDB request" });
    },

    async create(bucket) {
      const cfg = liveConfig("helpers.ObjectDB");
      if (!cfg.live) {
        return mockBucketsCreate(bucket);
      }
      return requestJSON("PUT", `/objectdb/v1/buckets/${encodePathPart(bucket)}`, cfg, { errorPrefix: "helpers.ObjectDB request" });
    },

    async exists(bucket) {
      const cfg = liveConfig("helpers.ObjectDB");
      if (!cfg.live) {
        return mockBucketsExists(bucket);
      }
      const result = await requestJSON("GET", "/objectdb/v1/buckets", cfg, { errorPrefix: "helpers.ObjectDB request" });
      const target = String(bucket);
      return !!(result && Array.isArray(result.buckets) && result.buckets.find((entry) => entry && entry.name === target));
    },

    async delete(bucket) {
      const cfg = liveConfig("helpers.ObjectDB");
      if (!cfg.live) {
        return mockBucketsDelete(bucket);
      }
      return requestJSON("DELETE", `/objectdb/v1/buckets/${encodePathPart(bucket)}`, cfg, { errorPrefix: "helpers.ObjectDB request" });
    },
  },

  keys: {
    async list(bucket, options = {}) {
      const cfg = liveConfig("helpers.ObjectDB");
      if (!cfg.live) {
        return mockKeysList(bucket, options);
      }
      return requestJSON("GET", `/objectdb/v1/buckets/${encodePathPart(bucket)}/keys`, cfg, {
        query: listOptions(options),
        errorPrefix: "helpers.ObjectDB request",
      });
    },
  },

  async put(bucket, key, value, options = {}) {
    const cfg = liveConfig("helpers.ObjectDB");
    if (!cfg.live) {
      return mockPut(bucket, key, value);
    }

    let body;
    let contentType = typeof options.contentType === "string" ? options.contentType : "";
    if (typeof value === "string") {
      body = value;
      if (contentType === "") {
        contentType = "text/plain; charset=utf-8";
      }
    } else if (value instanceof Uint8Array || value instanceof ArrayBuffer) {
      body = value;
      if (contentType === "") {
        contentType = "application/octet-stream";
      }
    } else {
      body = JSON.stringify(value);
      if (contentType === "") {
        contentType = "application/json";
      }
    }

    return requestJSON("PUT", `/objectdb/v1/buckets/${encodePathPart(bucket)}/values/${encodePathPart(key)}`, cfg, {
      body,
      headers: {
        "Content-Type": contentType,
      },
      errorPrefix: "helpers.ObjectDB request",
    });
  },

  async get(bucket, key, options = {}) {
    const cfg = liveConfig("helpers.ObjectDB");
    if (!cfg.live) {
      return mockGet(bucket, key);
    }

    const responseType = typeof options.type === "string" ? options.type : "json";
    const path = `/objectdb/v1/buckets/${encodePathPart(bucket)}/values/${encodePathPart(key)}`;
    const value = await requestJSON("GET", path, cfg, {
      query: responseType ? [["type", responseType]] : [],
      allowNotFound: true,
      expectText: responseType !== "json",
      errorPrefix: "helpers.ObjectDB request",
    });
    return value;
  },

  async head(bucket, key) {
    const cfg = liveConfig("helpers.ObjectDB");
    if (!cfg.live) {
      return mockHead(bucket, key);
    }

    const listed = await requestJSON("GET", `/objectdb/v1/buckets/${encodePathPart(bucket)}/keys`, cfg, {
      query: [["prefix", String(key)], ["limit", "1000"]],
      errorPrefix: "helpers.ObjectDB request",
    });
    if (!listed || !Array.isArray(listed.keys)) {
      return null;
    }
    const found = listed.keys.find((entry) => entry && entry.name === String(key));
    if (!found) {
      return null;
    }
    return found.metadata || null;
  },

  async delete(bucket, key) {
    const cfg = liveConfig("helpers.ObjectDB");
    if (!cfg.live) {
      return mockDelete(bucket, key);
    }
    return requestJSON("DELETE", `/objectdb/v1/buckets/${encodePathPart(bucket)}/values/${encodePathPart(key)}`, cfg, {
      errorPrefix: "helpers.ObjectDB request",
    });
  },
};

function mockMetricsQuery(query, options = {}) {
  const parsed = parsePromLikeQuery(query);
  const evalTimeMs = parseTimeInput(options.time) ?? Date.now();
  return {
    status: "success",
    data: buildMockVectorResult(parsed, evalTimeMs),
  };
}

function mockMetricsQueryRange(query, options = {}) {
  const parsed = parsePromLikeQuery(query);
  const startMs = parseTimeInput(options.start);
  const endMs = parseTimeInput(options.end);
  const stepMs = parseStepInput(options.step);
  if (startMs == null || endMs == null || !stepMs || stepMs <= 0) {
    throw new Error("helpers.Metrics.queryRange requires start, end, and step");
  }
  const matrix = new Map();
  for (let ts = startMs; ts <= endMs; ts += stepMs) {
    const vector = buildMockVectorResult(parsed, ts);
    for (const item of vector.result) {
      const key = JSON.stringify(item.metric);
      const entry = matrix.get(key) || { metric: item.metric, values: [] };
      entry.values.push(item.value);
      matrix.set(key, entry);
    }
  }
  return {
    status: "success",
    data: {
      resultType: "matrix",
      result: Array.from(matrix.values()),
    },
  };
}

function mockMetricsSeries(options = {}) {
  const rows = allMockMetricRows();
  const selectors = Array.isArray(options.match) ? options.match.map(parseSelector) : [];
  const out = new Map();
  for (const row of rows) {
    const labels = metricLabels(row.bucket, row.data);
    if (selectors.length > 0 && !selectors.some((selector) => matchesSelector(labels, selector))) {
      continue;
    }
    out.set(JSON.stringify(labels), labels);
  }
  return { status: "success", data: Array.from(out.values()) };
}

function mockMetricsLabels(options = {}) {
  const series = mockMetricsSeries(options).data;
  const labels = new Set();
  for (const item of series) {
    for (const key of Object.keys(item)) {
      labels.add(key);
    }
  }
  return { status: "success", data: Array.from(labels).sort() };
}

function mockMetricsLabelValues(name, options = {}) {
  const series = mockMetricsSeries(options).data;
  const values = new Set();
  for (const item of series) {
    if (typeof item[name] === "string") {
      values.add(item[name]);
    }
  }
  return { status: "success", data: Array.from(values).sort() };
}

function mockMetricsMetadataQuery(options = {}) {
  const metricName = typeof options.metric === "string" ? options.metric.trim() : "";
  const data = {};
  for (const [bucket, definitions] of mockMetricsMetadata.entries()) {
    for (const [name, definition] of Object.entries(definitions)) {
      if (metricName && metricName !== name) {
        continue;
      }
      if (!Array.isArray(data[name])) {
        data[name] = [];
      }
      data[name].push({
        type: definition.kind,
        help: definition.description,
        unit: definition.unit,
        bucket,
        labels: (definition.labels || []).join(","),
      });
    }
  }
  return { status: "success", data };
}

const helpersMetrics = {
  async query(query, options = {}) {
    const cfg = liveConfig("helpers.Metrics.query");
    if (!cfg.live) {
      return mockMetricsQuery(query, options);
    }
    return requestJSON("GET", "/metrics/v1/api/v1/query", cfg, {
      query: queryOptions({ query, time: options.time }),
      errorPrefix: "helpers.Metrics request",
    });
  },

  async queryRange(query, options = {}) {
    const cfg = liveConfig("helpers.Metrics.queryRange");
    if (!cfg.live) {
      return mockMetricsQueryRange(query, options);
    }
    return requestJSON("GET", "/metrics/v1/api/v1/query_range", cfg, {
      query: queryOptions({ query, start: options.start, end: options.end, step: options.step }),
      errorPrefix: "helpers.Metrics request",
    });
  },

  async series(options = {}) {
    const cfg = liveConfig("helpers.Metrics.series");
    if (!cfg.live) {
      return mockMetricsSeries(options);
    }
    return requestJSON("GET", "/metrics/v1/api/v1/series", cfg, {
      query: queryOptions({ "match[]": options.match, start: options.start, end: options.end }),
      errorPrefix: "helpers.Metrics request",
    });
  },

  async labels(options = {}) {
    const cfg = liveConfig("helpers.Metrics.labels");
    if (!cfg.live) {
      return mockMetricsLabels(options);
    }
    return requestJSON("GET", "/metrics/v1/api/v1/labels", cfg, {
      query: queryOptions({ "match[]": options.match, start: options.start, end: options.end }),
      errorPrefix: "helpers.Metrics request",
    });
  },

  async labelValues(name, options = {}) {
    const cfg = liveConfig("helpers.Metrics.labelValues");
    if (!cfg.live) {
      return mockMetricsLabelValues(name, options);
    }
    return requestJSON("GET", `/metrics/v1/api/v1/label/${encodePathPart(name)}/values`, cfg, {
      query: queryOptions({ "match[]": options.match, start: options.start, end: options.end }),
      errorPrefix: "helpers.Metrics request",
    });
  },

  async metadata(options = {}) {
    const cfg = liveConfig("helpers.Metrics.metadata");
    if (!cfg.live) {
      return mockMetricsMetadataQuery(options);
    }
    return requestJSON("GET", "/metrics/v1/api/v1/metadata", cfg, {
      query: queryOptions({ metric: options.metric }),
      errorPrefix: "helpers.Metrics request",
    });
  },
};

function instantiateMetrics(definition, bucketName) {
  if (!definition || typeof definition !== "object" || Array.isArray(definition)) {
    throw new Error("helpers.instantiateMetrics: definition must be an object");
  }

  const normalizedBucket = String(bucketName || "").trim();
  if (normalizedBucket === "") {
    throw new Error("helpers.instantiateMetrics: bucket name is required");
  }

  const instruments = {};
  const state = {
    bucket: normalizedBucket,
    definitions: normalizeMetricDefinitions(definition),
    buffer: new Map(),
    ensureStarted: false,
    ensurePromise: Promise.resolve(),
    flushPromise: Promise.resolve(),
  };
  metricsStates.add(state);

  for (const [metricName, metricDefinition] of Object.entries(state.definitions)) {
    const metricKind = metricDefinition.kind;

    if (metricKind === "counter") {
      instruments[metricName] = {
        inc(valueOrLabels = 1, maybeLabels = {}) {
          let amount = 1;
          let labels = maybeLabels;
          if (typeof valueOrLabels === "object" && valueOrLabels !== null && !Array.isArray(valueOrLabels)) {
            labels = valueOrLabels;
          } else {
            amount = Number(valueOrLabels);
          }
          if (!Number.isFinite(amount) || amount <= 0) {
            throw new Error(`helpers.instantiateMetrics: ${metricName}.inc value must be a positive number`);
          }
          recordMetric(state, metricName, metricDefinition, amount, labels);
        },
      };
      continue;
    }

    instruments[metricName] = {
      observe(value, labels = {}) {
        const amount = Number(value);
        if (!Number.isFinite(amount)) {
          throw new Error(`helpers.instantiateMetrics: ${metricName}.observe value must be a finite number`);
        }
        recordMetric(state, metricName, metricDefinition, amount, labels);
      },
    };
  }

  return instruments;
}

function recordMetric(state, metricName, metricDefinition, value, labelsInput) {
  ensureMetricsBucket(state);
  const windowStart = windowStartISOString(Date.now());
  const labels = normalizeMetricLabels(metricName, metricDefinition, labelsInput);
  const bufferKey = `${metricName}:${metricDefinition.kind}:${windowStart}:${JSON.stringify(labels)}`;
  let entry = state.buffer.get(bufferKey);
  if (!entry) {
    entry = {
      metric: metricName,
      kind: metricDefinition.kind,
      windowStart,
      labels,
      count: 0,
      sum: 0,
    };
    if (metricDefinition.kind === "histogram") {
      entry.values = [];
    }
    state.buffer.set(bufferKey, entry);
  }

  entry.count += 1;
  entry.sum += value;
  if (metricDefinition.kind === "histogram") {
    entry.values.push(value);
  }

  queueMetricsFlush(state, { flushCurrentWindow: false });
}

function ensureMetricsBucket(state) {
  if (state.ensureStarted) {
    return;
  }
  state.ensureStarted = true;

  const cfg = liveConfig("helpers.instantiateMetrics");
  if (!cfg.live) {
    mockMetricsBuckets.add(state.bucket);
    if (!mockMetricsRows.has(state.bucket)) {
      mockMetricsRows.set(state.bucket, []);
    }
    mockMetricsMetadata.set(state.bucket, state.definitions);
    state.ensurePromise = Promise.resolve();
    return;
  }

  state.ensurePromise = requestJSON("PUT", `/metrics/v1/buckets/${encodePathPart(state.bucket)}`, cfg, {
    errorPrefix: "helpers.Metrics request",
  }).then(() => requestJSON("PUT", `/metrics/v1/buckets/${encodePathPart(state.bucket)}/metadata`, cfg, {
    body: JSON.stringify({ metrics: state.definitions }),
    headers: { "Content-Type": "application/json" },
    errorPrefix: "helpers.Metrics request",
  }));
  queueWithContext(state.ensurePromise);
}

function queueMetricsFlush(state, options = {}) {
  state.flushPromise = state.flushPromise.then(async () => {
    await state.ensurePromise;
    const rows = collectMetricRows(state, options);
    if (rows.length === 0) {
      return;
    }

    const cfg = liveConfig("helpers.instantiateMetrics");
    if (!cfg.live) {
      const stored = mockMetricsRows.get(state.bucket) || [];
      stored.push(...rows);
      mockMetricsRows.set(state.bucket, stored);
      return;
    }

    await requestJSON("POST", `/metrics/v1/buckets/${encodePathPart(state.bucket)}/rows`, cfg, {
      body: JSON.stringify({
        rows: rows.map((row) => ({ ts: row.windowStart, data: row })),
      }),
      headers: { "Content-Type": "application/json" },
      errorPrefix: "helpers.Metrics request",
    });
  });

  queueWithContext(state.flushPromise);
}

function collectMetricRows(state, options = {}) {
  const flushCurrentWindow = options.flushCurrentWindow === true;
  const currentWindow = windowStartISOString(Date.now());
  const rows = [];

  for (const [bufferKey, entry] of Array.from(state.buffer.entries())) {
    if (!flushCurrentWindow && entry.windowStart >= currentWindow) {
      continue;
    }
    rows.push(cloneMetricRow(entry));
    state.buffer.delete(bufferKey);
  }

  rows.sort((left, right) => left.windowStart.localeCompare(right.windowStart) || left.metric.localeCompare(right.metric));
  return rows;
}

function cloneMetricRow(entry) {
  const row = {
    metric: entry.metric,
    kind: entry.kind,
    windowStart: entry.windowStart,
    labels: { ...entry.labels },
    count: entry.count,
    sum: entry.sum,
  };
  if (entry.kind === "histogram") {
    row.values = [...entry.values];
  }
  return row;
}

function queueFlushAllMetricsStates() {
  for (const state of metricsStates) {
    if (state.buffer.size === 0) {
      continue;
    }
    queueMetricsFlush(state, { flushCurrentWindow: true });
  }
}

async function settleAllMetricFlushes() {
  const flushes = [];
  for (const state of metricsStates) {
    flushes.push(Promise.resolve(state.flushPromise).catch(() => {}));
  }
  await Promise.all(flushes);
}

function queueWithContext(promise) {
  if (currentCtx && typeof currentCtx.waitUntil === "function") {
    currentCtx.waitUntil(Promise.resolve(promise).catch(() => {}));
  }
}

function windowStartISOString(timestampMs) {
  const windowStart = Math.floor(timestampMs / metricsWindowMs) * metricsWindowMs;
  return new Date(windowStart).toISOString();
}

export const helpers = {
  ObjectDB: helpersObjectDB,
  Metrics: helpersMetrics,
  instantiateMetrics,
};

function normalizeMetricDefinitions(definition) {
  const normalized = {};
  for (const [metricName, rawDefinition] of Object.entries(definition)) {
    if (typeof metricName !== "string" || metricName.trim() === "") {
      throw new Error("helpers.instantiateMetrics: metric names must be non-empty strings");
    }
    if (typeof rawDefinition === "string") {
      if (rawDefinition !== "counter" && rawDefinition !== "histogram") {
        throw new Error(`helpers.instantiateMetrics: unsupported metric kind for ${metricName}: ${rawDefinition}`);
      }
      normalized[metricName] = {
        kind: rawDefinition,
        description: metricName,
        unit: rawDefinition === "counter" ? "count" : "value",
        labels: [],
      };
      continue;
    }
    if (!rawDefinition || typeof rawDefinition !== "object" || Array.isArray(rawDefinition)) {
      throw new Error(`helpers.instantiateMetrics: metric definition for ${metricName} must be a string or object`);
    }
    const kind = rawDefinition.kind;
    if (kind !== "counter" && kind !== "histogram") {
      throw new Error(`helpers.instantiateMetrics: unsupported metric kind for ${metricName}: ${kind}`);
    }
    const labels = Array.isArray(rawDefinition.labels) ? rawDefinition.labels.map((label) => String(label)).sort() : [];
    normalized[metricName] = {
      kind,
      description: String(rawDefinition.description || metricName),
      unit: String(rawDefinition.unit || (kind === "counter" ? "count" : "value")),
      labels,
    };
  }
  return normalized;
}

function normalizeMetricLabels(metricName, metricDefinition, labelsInput) {
  if (!labelsInput || typeof labelsInput !== "object" || Array.isArray(labelsInput)) {
    if (typeof labelsInput === "undefined") {
      return {};
    }
    throw new Error(`helpers.instantiateMetrics: ${metricName} labels must be an object`);
  }
  const labels = {};
  for (const [key, value] of Object.entries(labelsInput)) {
    if (!metricDefinition.labels.includes(key)) {
      throw new Error(`helpers.instantiateMetrics: ${metricName} received undeclared label ${key}`);
    }
    if (value == null) {
      continue;
    }
    labels[key] = String(value);
  }
  return Object.fromEntries(Object.entries(labels).sort(([a], [b]) => a.localeCompare(b)));
}

function allMockMetricRows() {
  const rows = [];
  for (const [bucket, bucketRows] of mockMetricsRows.entries()) {
    for (const row of bucketRows) {
      rows.push({ bucket, data: row });
    }
  }
  return rows;
}

function buildMockVectorResult(parsed, evalTimeMs) {
  const rows = allMockMetricRows();
  const grouped = new Map();
  for (const row of rows) {
    const labels = metricLabels(row.bucket, row.data);
    if (!matchesSelector(labels, parsed.selector)) {
      continue;
    }
    const key = JSON.stringify(labels);
    const entry = grouped.get(key) || { metric: labels, points: [] };
    if (row.data.kind === "histogram") {
      for (const value of row.data.values || []) {
        entry.points.push({ ts: Date.parse(row.data.windowStart), value });
      }
    } else {
      entry.points.push({ ts: Date.parse(row.data.windowStart), value: row.data.sum });
    }
    grouped.set(key, entry);
  }

  const result = [];
  for (const entry of grouped.values()) {
    const value = evaluateMockSeries(parsed, entry.points, evalTimeMs);
    if (value == null) {
      continue;
    }
    result.push({ metric: entry.metric, value: [evalTimeMs / 1000, String(value)] });
  }
  return { resultType: "vector", result };
}

function evaluateMockSeries(parsed, points, evalTimeMs) {
  const lookbackMs = 5 * 60 * 1000;
  if (parsed.kind === "selector") {
    let latest = null;
    for (const point of points) {
      if (point.ts <= evalTimeMs && point.ts >= evalTimeMs - lookbackMs && (!latest || point.ts > latest.ts)) {
        latest = point;
      }
    }
    return latest ? latest.value : null;
  }
  const startMs = evalTimeMs - parsed.rangeMs;
  const window = points.filter((point) => point.ts >= startMs && point.ts <= evalTimeMs).map((point) => point.value);
  if (window.length === 0) {
    return null;
  }
  switch (parsed.fn) {
    case "sum_over_time":
    case "increase":
      return window.reduce((sum, value) => sum + value, 0);
    case "rate":
      return window.reduce((sum, value) => sum + value, 0) / (parsed.rangeMs / 1000);
    case "avg_over_time":
      return window.reduce((sum, value) => sum + value, 0) / window.length;
    case "p50":
      return percentile(window, 50);
    case "p90":
      return percentile(window, 90);
    case "p95":
      return percentile(window, 95);
    case "p99":
      return percentile(window, 99);
    default:
      return null;
  }
}

function percentile(values, pct) {
  if (values.length === 0) {
    return null;
  }
  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil((pct / 100) * sorted.length) - 1));
  return sorted[index];
}

function parsePromLikeQuery(rawQuery) {
  const query = String(rawQuery || "").trim();
  const fnMatch = query.match(/^([a-zA-Z_][a-zA-Z0-9_]*)\((.+)\)$/);
  if (fnMatch) {
    const inner = fnMatch[2].trim();
    const rangeMatch = inner.match(/^(.*)\[([^\]]+)\]$/);
    if (!rangeMatch) {
      throw new Error(`helpers.Metrics: invalid range query: ${query}`);
    }
    return { kind: "rangeFunction", fn: fnMatch[1], selector: parseSelector(rangeMatch[1].trim()), rangeMs: parseDurationInput(rangeMatch[2].trim()) };
  }
  return { kind: "selector", selector: parseSelector(query) };
}

function parseSelector(rawSelector) {
  const match = String(rawSelector || "").trim().match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{(.*)\})?$/);
  if (!match) {
    throw new Error(`helpers.Metrics: invalid selector: ${rawSelector}`);
  }
  const labels = { __name__: match[1] };
  const body = match[2] ? match[2].trim() : "";
  if (body) {
    for (const part of body.split(",")) {
      const matcher = part.trim().match(/^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"([^"]*)"$/);
      if (!matcher) {
        throw new Error(`helpers.Metrics: unsupported matcher: ${part.trim()}`);
      }
      labels[matcher[1]] = matcher[2];
    }
  }
  return { metric: match[1], matchers: labels };
}

function metricLabels(bucket, row) {
  return {
    __name__: row.metric,
    bucket,
    ...(row.labels || {}),
  };
}

function matchesSelector(labels, selector) {
  for (const [key, value] of Object.entries(selector.matchers)) {
    if (labels[key] !== value) {
      return false;
    }
  }
  return true;
}

function parseTimeInput(value) {
  if (value == null || String(value).trim() === "") {
    return null;
  }
  const raw = String(value).trim();
  const numeric = Number(raw);
  if (Number.isFinite(numeric)) {
    return numeric * 1000;
  }
  const parsed = Date.parse(raw);
  return Number.isNaN(parsed) ? null : parsed;
}

function parseStepInput(value) {
  if (value == null || String(value).trim() === "") {
    return null;
  }
  const raw = String(value).trim();
  const numeric = Number(raw);
  if (Number.isFinite(numeric)) {
    return numeric * 1000;
  }
  return parseDurationInput(raw);
}

function parseDurationInput(raw) {
  const units = { ms: 1, s: 1000, m: 60 * 1000, h: 60 * 60 * 1000, d: 24 * 60 * 60 * 1000 };
  let total = 0;
  let consumed = 0;
  const pattern = /(\d+(?:\.\d+)?)(ms|s|m|h|d)/g;
  let match;
  while ((match = pattern.exec(raw)) !== null) {
    total += Number(match[1]) * units[match[2]];
    consumed += match[0].length;
  }
  if (consumed !== raw.length || total <= 0) {
    throw new Error(`helpers.Metrics: invalid duration: ${raw}`);
  }
  return total;
}
