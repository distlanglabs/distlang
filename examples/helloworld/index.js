import { ObjectDB } from "distlang/core";

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const key = url.searchParams.get("key") || "hello/message";

    if (request.method === "POST") {
      const value = {
        message: "Hello from ObjectDB!",
        updatedAt: new Date().toISOString(),
      };
      await ObjectDB.put(key, value);
      return Response.json({ ok: true, key, value, source: "write" });
    }

    let value = await ObjectDB.get(key);
    if (value == null) {
      value = await ObjectDB.create(key, {
        message: "Hello Worker!",
        seeded: true,
      });
    }

    return Response.json({
      key,
      value,
      keys: await ObjectDB.list({ prefix: "hello/" }),
      source: env && env.DISTLANG_KV ? "cloudflare-kv" : "memory",
    });
  },
};
