import {
  appendLiveMetricRows,
  ensureLiveMetricsBucket,
  liveMetricsLabelValues,
  liveMetricsLabels,
  liveMetricsMetadata,
  liveMetricsQuery,
  liveMetricsQueryRange,
  liveMetricsSeries,
} from "./metrics_live.js";
import {
  appendMockMetricRows,
  ensureMockMetricsBucket,
  mockMetricsLabelValues,
  mockMetricsLabels,
  mockMetricsMetadataQuery,
  mockMetricsQuery,
  mockMetricsQueryRange,
  mockMetricsSeries,
} from "./metrics_mock.js";
import { liveConfig, queueWithContext } from "./shared.js";

const metricsStates = new Set();
const metricsWindowMs = 1 * 1000;

export const helpersMetrics = {
  async query(query, options = {}) {
    const cfg = liveConfig("helpers.Metrics.query");
    return cfg.live ? liveMetricsQuery(cfg, query, options) : mockMetricsQuery(query, options);
  },

  async queryRange(query, options = {}) {
    const cfg = liveConfig("helpers.Metrics.queryRange");
    return cfg.live ? liveMetricsQueryRange(cfg, query, options) : mockMetricsQueryRange(query, options);
  },

  async series(options = {}) {
    const cfg = liveConfig("helpers.Metrics.series");
    return cfg.live ? liveMetricsSeries(cfg, options) : mockMetricsSeries(options);
  },

  async labels(options = {}) {
    const cfg = liveConfig("helpers.Metrics.labels");
    return cfg.live ? liveMetricsLabels(cfg, options) : mockMetricsLabels(options);
  },

  async labelValues(name, options = {}) {
    const cfg = liveConfig("helpers.Metrics.labelValues");
    return cfg.live ? liveMetricsLabelValues(cfg, name, options) : mockMetricsLabelValues(name, options);
  },

  async metadata(options = {}) {
    const cfg = liveConfig("helpers.Metrics.metadata");
    return cfg.live ? liveMetricsMetadata(cfg, options) : mockMetricsMetadataQuery(options);
  },
};

export function instantiateMetrics(definition, bucketName) {
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
    if (metricDefinition.kind === "counter") {
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
    ensureMockMetricsBucket(state.bucket, state.definitions);
    state.ensurePromise = Promise.resolve();
    return;
  }

  state.ensurePromise = ensureLiveMetricsBucket(cfg, state.bucket, state.definitions);
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
      appendMockMetricRows(state.bucket, rows);
      return;
    }

    await appendLiveMetricRows(cfg, state.bucket, rows);
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

export function queueFlushAllMetricsStates() {
  for (const state of metricsStates) {
    if (state.buffer.size === 0) {
      continue;
    }
    queueMetricsFlush(state, { flushCurrentWindow: true });
  }
}

export async function settleAllMetricFlushes() {
  const flushes = [];
  for (const state of metricsStates) {
    flushes.push(Promise.resolve(state.flushPromise).catch(() => {}));
  }
  await Promise.all(flushes);
}

function windowStartISOString(timestampMs) {
  const windowStart = Math.floor(timestampMs / metricsWindowMs) * metricsWindowMs;
  return new Date(windowStart).toISOString();
}

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
