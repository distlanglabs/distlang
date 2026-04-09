const defaultStoreBaseURL = "https://api.distlang.com";

export const runtimeState = {
  currentEnv: null,
  currentCtx: null,
};

export function setRuntimeContext(env, ctx) {
  runtimeState.currentEnv = env || null;
  runtimeState.currentCtx = ctx || null;
}

export function clearRuntimeContext() {
  runtimeState.currentCtx = null;
}

export function envString(key) {
  if (!runtimeState.currentEnv) {
    return "";
  }
  const value = runtimeState.currentEnv[key];
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}

export function envService(key) {
  if (!runtimeState.currentEnv) {
    return null;
  }
  const value = runtimeState.currentEnv[key];
  if (!value || typeof value.fetch !== "function") {
    return null;
  }
  return value;
}

export function helpersMode() {
  const mode = envString("DISTLANG_HELPERS_MODE").toLowerCase();
  if (mode === "mock" || mode === "live") {
    return mode;
  }
  return "auto";
}

export function liveConfig(errorPrefix) {
  const mode = helpersMode();
  const token = envString("DISTLANG_SERVICE_TOKEN");
  const baseURL = (envString("DISTLANG_STORE_BASE_URL") || defaultStoreBaseURL).replace(/\/$/, "");
  const service = envService("DISTLANG_STORE");
  const shouldUseLive = mode === "live" || (mode === "auto" && token !== "");

  if (!shouldUseLive) {
    return { live: false };
  }
  if (token === "") {
    throw new Error(`${errorPrefix} requires DISTLANG_SERVICE_TOKEN in live mode`);
  }

  return { live: true, token, baseURL, service };
}

export function encodePathPart(value) {
  return encodeURIComponent(String(value));
}

export function listOptions(options = {}) {
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

export function queryOptions(options = {}) {
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

export async function requestJSON(method, path, cfg, options = {}) {
  const url = new URL(path, cfg.baseURL);
  if (Array.isArray(options.query)) {
    for (const [key, value] of options.query) {
      url.searchParams.append(key, value);
    }
  }

  const headers = {
    Authorization: `Bearer ${cfg.token}`,
    ...options.headers,
  };
  const request = new Request(url.toString(), {
    method,
    headers,
    body: options.body,
  });
  const res = cfg.service ? await cfg.service.fetch(request) : await fetch(request);

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

export function queueWithContext(promise) {
  if (runtimeState.currentCtx && typeof runtimeState.currentCtx.waitUntil === "function") {
    runtimeState.currentCtx.waitUntil(Promise.resolve(promise).catch(() => {}));
  }
}
