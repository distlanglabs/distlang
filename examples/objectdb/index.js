import { ObjectDB } from "distlang/core";

function sampleValue(key) {
  return {
    key,
    label: "ObjectDB example",
    updatedAt: new Date().toISOString(),
    tags: ["objectdb", "distlang"],
  };
}

function endpointIndex(baseUrl) {
  return {
    ok: true,
    name: "objectdb example",
    description: "Query-driven ObjectDB demo with raw JSON responses.",
    endpoints: [
      {
        method: "GET",
        path: "/",
        description: "List available endpoints when no key is provided.",
      },
      {
        method: "GET",
        path: "/?key=myKey",
        description: "Read an object. Seeds a sample object if the key is missing.",
      },
      {
        method: "POST",
        path: "/?key=myKey&value=hello",
        description: "Write a string value from the query string or a JSON value from the request body.",
      },
      {
        method: "DELETE",
        path: "/?key=myKey",
        description: "Delete an object.",
      },
    ],
    examples: {
      get: `${baseUrl}/?key=myKey`,
      post: `curl -X POST "${baseUrl}/?key=myKey&value=hello"`,
      postJson: `curl -X POST "${baseUrl}/?key=myKey" -H "Content-Type: application/json" -d '{"label":"from curl","count":1}'`,
      delete: `curl -X DELETE "${baseUrl}/?key=myKey"`,
    },
  };
}

async function requestValue(request, url, key) {
  if (url.searchParams.has("value")) {
    return url.searchParams.get("value");
  }
  if (request.headers.get("content-type")?.includes("application/json")) {
    const body = await request.json();
    if (body !== null && body !== undefined) {
      return body;
    }
  }
  return sampleValue(key);
}

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const pathname = url.pathname.replace(/\/$/, "") || "/";
    const key = url.searchParams.get("key") || "myKey";
    const source = env && env.DISTLANG_KV ? "cloudflare-kv" : "memory";

    if (pathname !== "/") {
      return Response.json(
        {
          ok: false,
          error: "Not Found",
          message: "Use GET / for the endpoint index or operate on /?key=...",
        },
        { status: 404 },
      );
    }

    if (request.method === "GET" && !url.searchParams.has("key")) {
      return Response.json(endpointIndex(url.origin));
    }

    if (!url.searchParams.has("key")) {
      return Response.json(
        {
          ok: false,
          error: "Missing key",
          message: "Pass ?key=... or use GET / to see the endpoint index.",
        },
        { status: 400 },
      );
    }

    if (request.method === "POST") {
      const value = await requestValue(request, url, key);
      await ObjectDB.put(key, value);
      return Response.json({
        ok: true,
        action: "put",
        key,
        value,
        keys: await ObjectDB.list(),
        source,
      });
    }

    if (request.method === "DELETE") {
      const deleted = await ObjectDB.delete(key);
      return Response.json({
        ok: true,
        action: "delete",
        key,
        deleted,
        keys: await ObjectDB.list(),
        source,
      });
    }

    let value = await ObjectDB.get(key);
    let seeded = false;
    if (value == null) {
      value = await ObjectDB.create(key, sampleValue(key));
      seeded = true;
    }

    return Response.json({
      ok: true,
      action: "get",
      key,
      value,
      seeded,
      keys: await ObjectDB.list(),
      hint: {
        postQuery: `fetch('/?key=${encodeURIComponent(key)}&value=hello-from-browser', { method: 'POST' })`,
        postJson: `fetch('/?key=${encodeURIComponent(key)}', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ label: 'from browser' }) })`,
        remove: `fetch('/?key=${encodeURIComponent(key)}', { method: 'DELETE' })`,
      },
      source,
    });
  },
};
