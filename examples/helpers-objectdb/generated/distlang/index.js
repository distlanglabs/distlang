import { InMemDB } from "distlang/core";

const defaultStoreBaseURL = "https://api.distlang.com";
let currentEnv = null;
let currentCtx = null;
const mockBuckets = new Set();
const mockMetricsBuckets = new Set();
const mockMetricsRows = new Map();
const metricsStates = new Set();
const metricsWindowMs = 1 * 1000;

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
    buffer: new Map(),
    ensureStarted: false,
    ensurePromise: Promise.resolve(),
    flushPromise: Promise.resolve(),
  };
  metricsStates.add(state);

  for (const [metricName, metricKind] of Object.entries(definition)) {
    if (typeof metricName !== "string" || metricName.trim() === "") {
      throw new Error("helpers.instantiateMetrics: metric names must be non-empty strings");
    }
    if (!isSupportedMetricKind(metricKind)) {
      throw new Error(`helpers.instantiateMetrics: unsupported metric kind for ${metricName}: ${metricKind}`);
    }

    if (metricKind === "counter") {
      instruments[metricName] = {
        inc(value = 1) {
          const amount = Number(value);
          if (!Number.isFinite(amount) || amount <= 0) {
            throw new Error(`helpers.instantiateMetrics: ${metricName}.inc value must be a positive number`);
          }
          recordMetric(state, metricName, metricKind, amount);
        },
      };
      continue;
    }

    if (metricKind === "gauge") {
      instruments[metricName] = {
        set(value) {
          const amount = Number(value);
          if (!Number.isFinite(amount)) {
            throw new Error(`helpers.instantiateMetrics: ${metricName}.set value must be a finite number`);
          }
          recordMetric(state, metricName, metricKind, amount);
        },
      };
      continue;
    }

    instruments[metricName] = {
      observe(value) {
        const amount = Number(value);
        if (!Number.isFinite(amount)) {
          throw new Error(`helpers.instantiateMetrics: ${metricName}.observe value must be a finite number`);
        }
        recordMetric(state, metricName, metricKind, amount);
      },
    };
  }

  return instruments;
}

function recordMetric(state, metricName, metricKind, value) {
  ensureMetricsBucket(state);
  const windowStart = windowStartISOString(Date.now());
  const bufferKey = `${metricName}:${metricKind}:${windowStart}`;
  let entry = state.buffer.get(bufferKey);
  if (!entry) {
    entry = {
      metric: metricName,
      kind: metricKind,
      windowStart,
      count: 0,
      sum: 0,
    };
    if (metricKind === "histogram") {
      entry.values = [];
    }
    state.buffer.set(bufferKey, entry);
  }

  if (metricKind === "gauge") {
    entry.count = 1;
    entry.sum = value;
  } else {
    entry.count += 1;
    entry.sum += value;
  }
  if (metricKind === "histogram") {
    entry.values.push(value);
  }

  queueMetricsFlush(state, { flushCurrentWindow: false });
}

function isSupportedMetricKind(kind) {
  return kind === "counter" || kind === "histogram" || kind === "gauge";
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
    state.ensurePromise = Promise.resolve();
    return;
  }

  state.ensurePromise = requestJSON("PUT", `/analyticsdb/v1/buckets/${encodePathPart(state.bucket)}`, cfg, {
    errorPrefix: "helpers.Metrics request",
  });
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

    await requestJSON("POST", `/analyticsdb/v1/buckets/${encodePathPart(state.bucket)}/rows`, cfg, {
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
  instantiateMetrics,
};
