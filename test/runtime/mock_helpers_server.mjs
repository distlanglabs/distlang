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

    if (req.method === "GET" && url.pathname === "/analyticsdb/v1") {
      json(res, 200, {
        ok: true,
        service: "analyticsdb",
        version: "mock",
        routes: {
          buckets: "/analyticsdb/v1/buckets/:bucket",
          rows: "/analyticsdb/v1/buckets/:bucket/rows",
        },
      });
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

    const analyticsBucketMatch = url.pathname.match(/^\/analyticsdb\/v1\/buckets\/([^/]+)$/);
    if (analyticsBucketMatch && req.method === "PUT") {
      const bucket = decodeURIComponent(analyticsBucketMatch[1]);
      const created = !analyticsBuckets.has(bucket);
      analyticsBuckets.add(bucket);
      if (!analyticsRows.has(bucket)) {
        analyticsRows.set(bucket, []);
      }
      json(res, 200, { ok: true, bucket, created });
      return;
    }

    const analyticsRowsMatch = url.pathname.match(/^\/analyticsdb\/v1\/buckets\/([^/]+)\/rows$/);
    if (analyticsRowsMatch && req.method === "POST") {
      const bucket = decodeURIComponent(analyticsRowsMatch[1]);
      if (!analyticsBuckets.has(bucket)) {
        analyticsBuckets.add(bucket);
      }
      if (!analyticsRows.has(bucket)) {
        analyticsRows.set(bucket, []);
      }
      const raw = await parseBody(req);
      const body = raw.length === 0 ? {} : JSON.parse(raw.toString("utf8"));
      const rows = Array.isArray(body.rows) ? body.rows : [];
      analyticsRows.get(bucket).push(...rows);
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
    analyticsRows,
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
