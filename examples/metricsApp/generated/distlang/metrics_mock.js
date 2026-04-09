const mockMetricSets = new Set();
const mockMetricsRows = new Map();
const mockMetricsMetadata = new Map();

export function ensureMockMetricsSet(metricSet, definitions) {
  mockMetricSets.add(metricSet);
  if (!mockMetricsRows.has(metricSet)) {
    mockMetricsRows.set(metricSet, []);
  }
  mockMetricsMetadata.set(metricSet, definitions);
}

export function appendMockMetricRows(metricSet, rows) {
  const stored = mockMetricsRows.get(metricSet) || [];
  stored.push(...rows);
  mockMetricsRows.set(metricSet, stored);
}

export function mockMetricsQuery(query, options = {}) {
  const parsed = parsePromLikeQuery(query);
  const evalTimeMs = parseTimeInput(options.time) ?? Date.now();
  return {
    status: "success",
    data: buildMockVectorResult(parsed, evalTimeMs),
  };
}

export function mockMetricsQueryRange(query, options = {}) {
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

export function mockMetricsSeries(options = {}) {
  const rows = allMockMetricRows();
  const selectors = Array.isArray(options.match) ? options.match.map(parseSelector) : [];
  const out = new Map();
  for (const row of rows) {
    const labels = metricLabels(row.metricSet, row.data);
    if (selectors.length > 0 && !selectors.some((selector) => matchesSelector(labels, selector))) {
      continue;
    }
    out.set(JSON.stringify(labels), labels);
  }
  return { status: "success", data: Array.from(out.values()) };
}

export function mockMetricsLabels(options = {}) {
  const series = mockMetricsSeries(options).data;
  const labels = new Set();
  for (const item of series) {
    for (const key of Object.keys(item)) {
      labels.add(key);
    }
  }
  return { status: "success", data: Array.from(labels).sort() };
}

export function mockMetricsLabelValues(name, options = {}) {
  const series = mockMetricsSeries(options).data;
  const values = new Set();
  for (const item of series) {
    if (typeof item[name] === "string") {
      values.add(item[name]);
    }
  }
  return { status: "success", data: Array.from(values).sort() };
}

export function mockMetricsMetadataQuery(options = {}) {
  const metricName = typeof options.metric === "string" ? options.metric.trim() : "";
  const data = {};
  for (const [metricSet, definitions] of mockMetricsMetadata.entries()) {
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
          metricSet,
          labels: (definition.labels || []).join(","),
        });
    }
  }
  return { status: "success", data };
}

function allMockMetricRows() {
  const rows = [];
  for (const [metricSet, metricSetRows] of mockMetricsRows.entries()) {
    for (const row of metricSetRows) {
      rows.push({ metricSet, data: row });
    }
  }
  return rows;
}

function buildMockVectorResult(parsed, evalTimeMs) {
  const rows = allMockMetricRows();
  const grouped = new Map();
  for (const row of rows) {
    const labels = metricLabels(row.metricSet, row.data);
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

function metricLabels(metricSet, row) {
  return {
    __name__: row.metric,
    metricSet,
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
