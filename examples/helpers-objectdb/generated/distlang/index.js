import { InMemDB } from "distlang/core";

const defaultStoreBaseURL = "https://api.distlang.com";
let currentEnv = null;
const mockBuckets = new Set();

export function wrapWorkerWithHelpers(worker) {
  return {
    ...worker,
    async fetch(request, env, ctx) {
      currentEnv = env || null;
      return worker.fetch(request, env, ctx);
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

function liveConfig() {
  const mode = helpersMode();
  const token = envString("DISTLANG_SERVICE_TOKEN");
  const baseURL = (envString("DISTLANG_STORE_BASE_URL") || defaultStoreBaseURL).replace(/\/$/, "");
  const shouldUseLive = mode === "live" || (mode === "auto" && token !== "");

  if (!shouldUseLive) {
    return { live: false };
  }
  if (token === "") {
    throw new Error("helpers.ObjectDB requires DISTLANG_SERVICE_TOKEN in live mode");
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
    throw new Error(`helpers.ObjectDB request failed (${res.status}): ${message}`);
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
    const cfg = liveConfig();
    if (!cfg.live) {
      return mockStatus();
    }
    return requestJSON("GET", "/objectdb/v1", cfg);
  },

  buckets: {
    async list() {
      const cfg = liveConfig();
      if (!cfg.live) {
        return mockBucketsList();
      }
      return requestJSON("GET", "/objectdb/v1/buckets", cfg);
    },

    async create(bucket) {
      const cfg = liveConfig();
      if (!cfg.live) {
        return mockBucketsCreate(bucket);
      }
      return requestJSON("PUT", `/objectdb/v1/buckets/${encodePathPart(bucket)}`, cfg);
    },

    async exists(bucket) {
      const cfg = liveConfig();
      if (!cfg.live) {
        return mockBucketsExists(bucket);
      }
      const result = await requestJSON("GET", "/objectdb/v1/buckets", cfg);
      const target = String(bucket);
      return !!(result && Array.isArray(result.buckets) && result.buckets.find((entry) => entry && entry.name === target));
    },

    async delete(bucket) {
      const cfg = liveConfig();
      if (!cfg.live) {
        return mockBucketsDelete(bucket);
      }
      return requestJSON("DELETE", `/objectdb/v1/buckets/${encodePathPart(bucket)}`, cfg);
    },
  },

  keys: {
    async list(bucket, options = {}) {
      const cfg = liveConfig();
      if (!cfg.live) {
        return mockKeysList(bucket, options);
      }
      return requestJSON("GET", `/objectdb/v1/buckets/${encodePathPart(bucket)}/keys`, cfg, {
        query: listOptions(options),
      });
    },
  },

  async put(bucket, key, value, options = {}) {
    const cfg = liveConfig();
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
    });
  },

  async get(bucket, key, options = {}) {
    const cfg = liveConfig();
    if (!cfg.live) {
      return mockGet(bucket, key);
    }

    const responseType = typeof options.type === "string" ? options.type : "json";
    const path = `/objectdb/v1/buckets/${encodePathPart(bucket)}/values/${encodePathPart(key)}`;
    const value = await requestJSON("GET", path, cfg, {
      query: responseType ? [["type", responseType]] : [],
      allowNotFound: true,
      expectText: responseType !== "json",
    });
    return value;
  },

  async head(bucket, key) {
    const cfg = liveConfig();
    if (!cfg.live) {
      return mockHead(bucket, key);
    }

    const listed = await requestJSON("GET", `/objectdb/v1/buckets/${encodePathPart(bucket)}/keys`, cfg, {
      query: [["prefix", String(key)], ["limit", "1000"]],
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
    const cfg = liveConfig();
    if (!cfg.live) {
      return mockDelete(bucket, key);
    }
    return requestJSON("DELETE", `/objectdb/v1/buckets/${encodePathPart(bucket)}/values/${encodePathPart(key)}`, cfg);
  },
};

export const helpers = {
  ObjectDB: helpersObjectDB,
};
