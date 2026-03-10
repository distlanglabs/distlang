const __distlangMemoryKV = new Map();
let __distlangCurrentKV = null;

function __distlangClone(value) {
  if (value == null) {
    return value;
  }
  return JSON.parse(JSON.stringify(value));
}

function __distlangUseBinding(binding) {
  __distlangCurrentKV = binding || null;
}

async function __distlangKVGet(key) {
  if (__distlangCurrentKV && typeof __distlangCurrentKV.get === "function") {
    const raw = await __distlangCurrentKV.get(key, "text");
    if (raw == null) {
      return null;
    }
    return JSON.parse(raw);
  }
  if (!__distlangMemoryKV.has(key)) {
    return null;
  }
  return __distlangClone(__distlangMemoryKV.get(key));
}

async function __distlangKVPut(key, value) {
  if (__distlangCurrentKV && typeof __distlangCurrentKV.put === "function") {
    await __distlangCurrentKV.put(key, JSON.stringify(value));
    return __distlangClone(value);
  }
  __distlangMemoryKV.set(key, __distlangClone(value));
  return __distlangClone(value);
}

async function __distlangKVDelete(key) {
  if (__distlangCurrentKV && typeof __distlangCurrentKV.delete === "function") {
    await __distlangCurrentKV.delete(key);
    return true;
  }
  return __distlangMemoryKV.delete(key);
}

async function __distlangKVList(options = {}) {
  const prefix = typeof options.prefix === "string" ? options.prefix : "";
  if (__distlangCurrentKV && typeof __distlangCurrentKV.list === "function") {
    const result = await __distlangCurrentKV.list({ prefix });
    const keys = result && Array.isArray(result.keys) ? result.keys : [];
    return keys.map((entry) => entry.name);
  }
  return Array.from(__distlangMemoryKV.keys()).filter((key) => key.startsWith(prefix));
}

globalThis.__distlangWrapDefault = function(worker) {
  return {
    ...worker,
    async fetch(request, env, ctx) {
      __distlangUseBinding(env && env.DISTLANG_KV);
      return worker.fetch(request, env, ctx);
    }
  };
};

export const ObjectDB = {
  async create(key, value) {
    const existing = await __distlangKVGet(key);
    if (existing != null) {
      throw new Error("ObjectDB.create: key already exists");
    }
    return __distlangKVPut(key, value);
  },
  async get(key) {
    return __distlangKVGet(key);
  },
  async put(key, value) {
    return __distlangKVPut(key, value);
  },
  async update(key, updater) {
    const current = await __distlangKVGet(key);
    const next = typeof updater === "function" ? await updater(current) : updater;
    return __distlangKVPut(key, next);
  },
  async delete(key) {
    return __distlangKVDelete(key);
  },
  async list(options) {
    return __distlangKVList(options);
  }
};
