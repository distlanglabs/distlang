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

function compileRoute(method, path, handlerSetName, routesByMethod) {
  if (typeof path !== "string" || path.trim() === "") {
    throw new Error(`simpleApp.instantiate: ${handlerSetName}.${method} route path must be a non-empty string`);
  }
  if (typeof routesByMethod[path] !== "function") {
    throw new Error(`simpleApp.instantiate: ${handlerSetName}.${method} route ${path} must map to a function`);
  }

  const normalizedPath = normalizePath(path);
  const segments = normalizedPath === "/" ? [] : normalizedPath.slice(1).split("/");
  const paramNames = [];
  for (const segment of segments) {
    if (segment.startsWith(":")) {
      const name = segment.slice(1).trim();
      if (name === "") {
        throw new Error(`simpleApp.instantiate: ${handlerSetName}.${method} route ${path} has an empty param name`);
      }
      paramNames.push(name);
    }
  }

  return {
    method,
    path: normalizedPath,
    segments,
    handler: routesByMethod[path],
    hasParams: paramNames.length > 0,
  };
}

function normalizeHandlerSet(handlerSetName, handlerSet) {
  if (!isPlainObject(handlerSet)) {
    throw new Error(`simpleApp.instantiate: ${handlerSetName} must be an object`);
  }
  if (!isPlainObject(handlerSet.routes)) {
    throw new Error(`simpleApp.instantiate: ${handlerSetName}.routes must be an object`);
  }

  const seen = new Set();
  const compiledByMethod = new Map();
  const routes = handlerSet.routes;

  for (const key of Object.keys(routes)) {
    const method = key.toUpperCase();
    if (!allowedMethods.has(method)) {
      throw new Error(`simpleApp.instantiate: ${handlerSetName}.routes has unsupported method ${key}`);
    }

    const routesByMethod = routes[key];
    if (!isPlainObject(routesByMethod)) {
      throw new Error(`simpleApp.instantiate: ${handlerSetName}.routes.${key} must be an object`);
    }

    const entries = compiledByMethod.get(method) || [];
    for (const path of Object.keys(routesByMethod)) {
      const compiled = compileRoute(method, path, handlerSetName, routesByMethod);
      const dedupeKey = `${method} ${compiled.path}`;
      if (seen.has(dedupeKey)) {
        throw new Error(`simpleApp.instantiate: duplicate route ${dedupeKey} in ${handlerSetName}`);
      }
      seen.add(dedupeKey);
      entries.push(compiled);
    }
    compiledByMethod.set(method, entries);
  }

  return {
    name: handlerSetName,
    routes: compiledByMethod,
    hasRootGetRoute: seen.has("GET /"),
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

  return {
    params,
    hasParams: route.hasParams,
  };
}

function resolveTarget() {
  const raw = globalThis && typeof globalThis.__DISTLANG_SIMPLEAPP_TARGET__ === "string"
    ? globalThis.__DISTLANG_SIMPLEAPP_TARGET__.trim()
    : "";

  if (raw === "" || raw === "handlerSet1") {
    return "handlerSet1";
  }
  if (raw === "handlerSet2") {
    return "handlerSet2";
  }

  throw new Error(`simpleApp.instantiate: unsupported handler set target ${raw}`);
}

function normalizeDB(db) {
  if (!isPlainObject(db) && typeof db !== "function") {
    throw new Error("simpleApp.instantiate: db must be an object");
  }
  return db;
}

function dbName(db) {
  if (
    db &&
    typeof db.status === "function" &&
    db.buckets &&
    typeof db.buckets.create === "function" &&
    db.keys &&
    typeof db.keys.list === "function"
  ) {
    return "ObjectDB";
  }
  return "unknown";
}

function dbMode(env) {
  const mode = env && typeof env.DISTLANG_HELPERS_MODE === "string"
    ? env.DISTLANG_HELPERS_MODE.trim().toLowerCase()
    : "";

  if (mode === "live") {
    return "live";
  }
  if (mode === "mock") {
    return "mock";
  }

  const token = env && typeof env.DISTLANG_SERVICE_TOKEN === "string"
    ? env.DISTLANG_SERVICE_TOKEN.trim()
    : "";
  return token === "" ? "mock" : "live";
}

function indexPayload(compiled, db, env, request) {
  return {
    ok: true,
    app: "simpleApp",
    activeHandlerSet: compiled.name,
    routes: routeIndex(compiled),
    routeExamples: routeIndexWithExamples(compiled, request),
    db: {
      name: dbName(db),
      mode: dbMode(env),
    },
  };
}

function buildFetchHandler(compiled, db) {
  return async function fetch(request, env, ctx) {
    const method = (request.method || "GET").toUpperCase();
    const pathname = new URL(request.url).pathname;
    const candidates = compiled.routes.get(method) || [];

    for (const route of candidates) {
      const matched = routeMatch(route, pathname);
      if (!matched) {
        continue;
      }

      const input = {
        req: request,
        db,
      };
      if (matched.hasParams) {
        input.params = matched.params;
      }

      const result = await route.handler(input, env, ctx);
      if (result instanceof Response) {
        return result;
      }
      if (result === undefined) {
        return new Response("", { status: 204 });
      }
      return Response.json(result);
    }

    if (method === "GET" && pathname === "/" && !compiled.hasRootGetRoute) {
      return Response.json(indexPayload(compiled, db, env, request));
    }

    return new Response("Not Found", { status: 404 });
  };
}

export const simpleApp = {
  instantiate(handlerSet1, handlerSet2, db) {
    const compiledSet1 = normalizeHandlerSet("handlerSet1", handlerSet1);
    const compiledSet2 = normalizeHandlerSet("handlerSet2", handlerSet2);
    const resolvedDB = normalizeDB(db);
    const target = resolveTarget();
    const active = target === "handlerSet2" ? compiledSet2 : compiledSet1;
    const activeFetch = buildFetchHandler(active, resolvedDB);

    return {
      async fetch(request, env, ctx) {
        return activeFetch(request, env, ctx);
      },
    };
  },
};
