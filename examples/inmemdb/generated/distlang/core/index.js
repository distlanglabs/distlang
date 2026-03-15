const memoryStore = new Map();
let currentBinding = null;

function cloneValue(value) {
  if (value == null) {
    return value;
  }
  return JSON.parse(JSON.stringify(value));
}

function useBinding(binding) {
  currentBinding = binding || null;
}

function hasBindingMethod(name) {
  return currentBinding && typeof currentBinding[name] === "function";
}

async function readValue(key) {
  if (hasBindingMethod("get")) {
    const raw = await currentBinding.get(key, "text");
    if (raw == null) {
      return null;
    }
    return JSON.parse(raw);
  }
  if (!memoryStore.has(key)) {
    return null;
  }
  return cloneValue(memoryStore.get(key));
}

async function writeValue(key, value) {
  if (hasBindingMethod("put")) {
    await currentBinding.put(key, JSON.stringify(value));
    return cloneValue(value);
  }
  memoryStore.set(key, cloneValue(value));
  return cloneValue(value);
}

async function deleteValue(key) {
  if (hasBindingMethod("delete")) {
    await currentBinding.delete(key);
    return true;
  }
  return memoryStore.delete(key);
}

async function listKeys(options = {}) {
  const prefix = typeof options.prefix === "string" ? options.prefix : "";
  if (hasBindingMethod("list")) {
    const result = await currentBinding.list({ prefix });
    const keys = result && Array.isArray(result.keys) ? result.keys : [];
    return keys.map((entry) => entry.name);
  }
  return Array.from(memoryStore.keys()).filter((key) => key.startsWith(prefix));
}

export function wrapWorkerWithInMemDB(worker) {
  return {
    ...worker,
    async fetch(request, env, ctx) {
      useBinding(env && env.DISTLANG_KV);
      return worker.fetch(request, env, ctx);
    },
  };
}

export const InMemDB = {
  async create(key, value) {
    const existing = await readValue(key);
    if (existing != null) {
      throw new Error("InMemDB.create: key already exists");
    }
    return writeValue(key, value);
  },
  async get(key) {
    return readValue(key);
  },
  async put(key, value) {
    return writeValue(key, value);
  },
  async update(key, updater) {
    const current = await readValue(key);
    const next = typeof updater === "function" ? await updater(current) : updater;
    return writeValue(key, next);
  },
  async delete(key) {
    return deleteValue(key);
  },
  async list(options) {
    return listKeys(options);
  },
};
