import { InMemDB } from "distlang/core";

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const key = url.searchParams.get("key") || "hello/message";

    if (request.method === "POST") {
      const value = {
        message: "Hello from InMemDB!",
        updatedAt: new Date().toISOString(),
      };
      await InMemDB.put(key, value);
      return Response.json({ ok: true, key, value, source: "write" });
    }

    let value = await InMemDB.get(key);
    if (value == null) {
      value = await InMemDB.create(key, {
        message: "Hello Worker!",
        seeded: true,
      });
    }

    return Response.json({
      key,
      value,
      keys: await InMemDB.list({ prefix: "hello/" }),
      source: env && env.DISTLANG_KV ? "cloudflare-kv" : "memory",
    });
  },
};
