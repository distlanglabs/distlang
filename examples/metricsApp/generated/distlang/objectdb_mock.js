import { InMemDB } from "distlang/core";

const mockBuckets = new Set();

function bucketKey(bucket, key) {
  return `${bucket}:${key}`;
}

export async function mockStatus() {
  return {
    ok: true,
    service: "objectdb",
    mode: "mock",
  };
}

export async function mockBucketsList() {
  return {
    ok: true,
    buckets: Array.from(mockBuckets).sort().map((name) => ({ name })),
  };
}

export async function mockBucketsCreate(bucket) {
  const name = String(bucket);
  const created = !mockBuckets.has(name);
  mockBuckets.add(name);
  return {
    ok: true,
    bucket: name,
    created,
  };
}

export async function mockBucketsExists(bucket) {
  return mockBuckets.has(String(bucket));
}

export async function mockBucketsDelete(bucket) {
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

export async function mockKeysList(bucket, options = {}) {
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

export async function mockPut(bucket, key, value) {
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

export async function mockGet(bucket, key) {
  const bucketName = String(bucket);
  if (!mockBuckets.has(bucketName)) {
    return null;
  }
  return InMemDB.get(bucketKey(bucketName, key));
}

export async function mockHead(bucket, key) {
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

export async function mockDelete(bucket, key) {
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
