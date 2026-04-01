import { encodePathPart, listOptions, requestJSON } from "./shared.js";

export async function liveStatus(cfg) {
  return requestJSON("GET", "/objectdb/v1", cfg, { errorPrefix: "helpers.ObjectDB request" });
}

export async function liveBucketsList(cfg) {
  return requestJSON("GET", "/objectdb/v1/buckets", cfg, { errorPrefix: "helpers.ObjectDB request" });
}

export async function liveBucketsCreate(cfg, bucket) {
  return requestJSON("PUT", `/objectdb/v1/buckets/${encodePathPart(bucket)}`, cfg, { errorPrefix: "helpers.ObjectDB request" });
}

export async function liveBucketsExists(cfg, bucket) {
  const result = await requestJSON("GET", "/objectdb/v1/buckets", cfg, { errorPrefix: "helpers.ObjectDB request" });
  const target = String(bucket);
  return !!(result && Array.isArray(result.buckets) && result.buckets.find((entry) => entry && entry.name === target));
}

export async function liveBucketsDelete(cfg, bucket) {
  return requestJSON("DELETE", `/objectdb/v1/buckets/${encodePathPart(bucket)}`, cfg, { errorPrefix: "helpers.ObjectDB request" });
}

export async function liveKeysList(cfg, bucket, options = {}) {
  return requestJSON("GET", `/objectdb/v1/buckets/${encodePathPart(bucket)}/keys`, cfg, {
    query: listOptions(options),
    errorPrefix: "helpers.ObjectDB request",
  });
}

export async function livePut(cfg, bucket, key, value, options = {}) {
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
    headers: { "Content-Type": contentType },
    errorPrefix: "helpers.ObjectDB request",
  });
}

export async function liveGet(cfg, bucket, key, options = {}) {
  const responseType = typeof options.type === "string" ? options.type : "json";
  const path = `/objectdb/v1/buckets/${encodePathPart(bucket)}/values/${encodePathPart(key)}`;
  return requestJSON("GET", path, cfg, {
    query: responseType ? [["type", responseType]] : [],
    allowNotFound: true,
    expectText: responseType !== "json",
    errorPrefix: "helpers.ObjectDB request",
  });
}

export async function liveHead(cfg, bucket, key) {
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
}

export async function liveDelete(cfg, bucket, key) {
  return requestJSON("DELETE", `/objectdb/v1/buckets/${encodePathPart(bucket)}/values/${encodePathPart(key)}`, cfg, {
    errorPrefix: "helpers.ObjectDB request",
  });
}
