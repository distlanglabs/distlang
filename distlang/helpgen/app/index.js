const allowedMethods = new Set(["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]);

function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function normalizePath(pathname) {
  if (typeof pathname !== "string" || pathname === "") {
    return "/";
  }
  if (pathname === "/") {
    return "/";
  }
  let value = pathname;
  if (!value.startsWith("/")) {
    value = `/${value}`;
  }
  return value.replace(/\/+$/, "") || "/";
}

function compileRoute(method, path, routesByMethod) {
  if (typeof path !== "string" || path.trim() === "") {
    throw new Error(`app: ${method} route path must be a non-empty string`);
  }
  if (typeof routesByMethod[path] !== "function") {
    throw new Error(`app: ${method} route ${path} must map to a function`);
  }

  const normalizedPath = normalizePath(path);
  const segments = normalizedPath === "/" ? [] : normalizedPath.slice(1).split("/");
  const paramNames = [];
  for (const segment of segments) {
    if (segment.startsWith(":")) {
      const name = segment.slice(1).trim();
      if (name === "") {
        throw new Error(`app: ${method} route ${path} has an empty param name`);
      }
      paramNames.push(name);
    }
  }

  return {
    method,
    path: normalizedPath,
    segments,
    handler: routesByMethod[path],
  };
}

function normalizeDBs(dbs) {
  if (!isPlainObject(dbs)) {
    throw new Error("app: state.dbs must be an object");
  }

  const resolved = {};
  for (const key of Object.keys(dbs)) {
    const value = dbs[key];
    if (!isPlainObject(value) && typeof value !== "function") {
      throw new Error(`app: state.dbs.${key} must be an object`);
    }
    resolved[key] = value;
  }
  return resolved;
}

function normalizeHandlers(handlers) {
  if (!isPlainObject(handlers)) {
    throw new Error("app: compute.handlers must be an object");
  }
  if (!isPlainObject(handlers.routes)) {
    throw new Error("app: compute.handlers.routes must be an object");
  }

  const seen = new Set();
  const compiledByMethod = new Map();
  const routes = handlers.routes;

  for (const key of Object.keys(routes)) {
    const method = key.toUpperCase();
    if (!allowedMethods.has(method)) {
      throw new Error(`app: compute.handlers.routes has unsupported method ${key}`);
    }

    const routesByMethod = routes[key];
    if (!isPlainObject(routesByMethod)) {
      throw new Error(`app: compute.handlers.routes.${key} must be an object`);
    }

    const entries = compiledByMethod.get(method) || [];
    for (const path of Object.keys(routesByMethod)) {
      const compiled = compileRoute(method, path, routesByMethod);
      const dedupeKey = `${method} ${compiled.path}`;
      if (seen.has(dedupeKey)) {
        throw new Error(`app: duplicate route ${dedupeKey}`);
      }
      seen.add(dedupeKey);
      entries.push(compiled);
    }
    compiledByMethod.set(method, entries);
  }

  return {
    routes: compiledByMethod,
    hasRootGetRoute: seen.has("GET /"),
  };
}

function normalizeSpec(spec) {
  if (!isPlainObject(spec)) {
    throw new Error("app: spec must be an object");
  }
  if (!isPlainObject(spec.state)) {
    throw new Error("app: state must be an object");
  }
  if (!isPlainObject(spec.compute)) {
    throw new Error("app: compute must be an object");
  }
  if ("uses" in spec.compute) {
    throw new Error("app: compute.uses is not supported yet");
  }
  if ("triggers" in spec.compute) {
    throw new Error("app: compute.triggers is not supported yet");
  }

  return {
    state: {
      dbs: normalizeDBs(spec.state.dbs),
    },
    compute: normalizeHandlers(spec.compute.handlers),
  };
}

function routeIndex(compiled) {
  const out = {};
  const methods = Array.from(compiled.routes.keys()).sort();
  for (const method of methods) {
    const entries = compiled.routes.get(method) || [];
    out[method] = entries.map((entry) => entry.path);
  }
  return out;
}

function pathForCurl(pathname) {
  const parts = pathname.split("/");
  const converted = parts.map((segment) => {
    if (segment.startsWith(":")) {
      return "sample";
    }
    return segment;
  });
  return converted.join("/") || "/";
}

function curlExample(origin, method, path) {
  const safeOrigin = typeof origin === "string" && origin.trim() !== "" ? origin : "http://127.0.0.1:5656";
  const target = `${safeOrigin}${pathForCurl(path)}`;

  if (method === "POST" || method === "PUT" || method === "PATCH") {
    return `curl -X ${method} \"${target}\" -H \"Content-Type: application/json\" -d '{}'`;
  }
  if (method === "DELETE") {
    return `curl -X DELETE \"${target}\"`;
  }

  return `curl \"${target}\"`;
}

function routeIndexWithExamples(compiled, request) {
  const out = {};
  const methods = Array.from(compiled.routes.keys()).sort();
  const origin = request && request.url ? new URL(request.url).origin : "http://127.0.0.1:5656";

  for (const method of methods) {
    const entries = compiled.routes.get(method) || [];
    out[method] = entries.map((entry) => ({
      path: entry.path,
      curl: curlExample(origin, method, entry.path),
    }));
  }

  return out;
}

function routeMatch(route, pathname) {
  const target = normalizePath(pathname);
  const pathSegments = target === "/" ? [] : target.slice(1).split("/");
  if (pathSegments.length !== route.segments.length) {
    return null;
  }

  const params = {};
  for (let index = 0; index < route.segments.length; index += 1) {
    const routeSegment = route.segments[index];
    const pathSegment = pathSegments[index];
    if (routeSegment.startsWith(":")) {
      params[routeSegment.slice(1)] = decodeURIComponent(pathSegment);
      continue;
    }
    if (routeSegment !== pathSegment) {
      return null;
    }
  }

  return params;
}

function indexPayload(compiled, state, request) {
  return {
    ok: true,
    app: "app",
    state: {
      dbs: Object.keys(state.dbs).sort(),
    },
    routes: routeIndex(compiled),
    routeExamples: routeIndexWithExamples(compiled, request),
  };
}

function buildFetchHandler(compiled, state) {
  return async function fetch(request, env, ctx) {
    const method = (request.method || "GET").toUpperCase();
    const pathname = new URL(request.url).pathname;
    const candidates = compiled.routes.get(method) || [];

    for (const route of candidates) {
      const params = routeMatch(route, pathname);
      if (params === null) {
        continue;
      }

      const result = await route.handler({ req: request, state, params }, env, ctx);
      if (result instanceof Response) {
        return result;
      }
      if (result === undefined) {
        return new Response("", { status: 204 });
      }
      return Response.json(result);
    }

    if (method === "GET" && pathname === "/" && !compiled.hasRootGetRoute) {
      return Response.json(indexPayload(compiled, state, request));
    }

    return new Response("Not Found", { status: 404 });
  };
}

export function app(spec) {
  const normalized = normalizeSpec(spec);
  const activeFetch = buildFetchHandler(normalized.compute, normalized.state);

  return {
    async fetch(request, env, ctx) {
      return activeFetch(request, env, ctx);
    },
  };
}
