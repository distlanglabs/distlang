import { helpers } from "distlang";

function json(data, init = {}) {
  return Response.json(data, init);
}

function notFound() {
  return json(
    {
      ok: false,
      error: "not_found",
      message: "Use /, /bucket/:bucket, or /bucket/:bucket/key/:key",
    },
    { status: 404 },
  );
}

async function parseValue(request) {
  const contentType = request.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    return request.json();
  }
  const text = await request.text();
  if (text.trim() === "") {
    return null;
  }
  return text;
}

export default {
  async fetch(request) {
    const url = new URL(request.url);
    const path = url.pathname.replace(/\/+$/, "") || "/";
    const segments = path.split("/").filter(Boolean);

    if (path === "/" && request.method === "GET") {
      const status = await helpers.ObjectDB.status();
      return json({
        ok: true,
        status,
        routes: {
          createBucket: "PUT /bucket/:bucket",
          listKeys: "GET /bucket/:bucket",
          putValue: "PUT /bucket/:bucket/key/:key",
          getValue: "GET /bucket/:bucket/key/:key",
          deleteValue: "DELETE /bucket/:bucket/key/:key",
        },
      });
    }

    if (segments.length === 2 && segments[0] === "bucket") {
      const bucket = decodeURIComponent(segments[1]);
      if (request.method === "PUT") {
        const result = await helpers.ObjectDB.buckets.create(bucket);
        return json({ ok: true, result });
      }
      if (request.method === "GET") {
        const keys = await helpers.ObjectDB.keys.list(bucket);
        return json({ ok: true, bucket, keys });
      }
      return notFound();
    }

    if (segments.length === 4 && segments[0] === "bucket" && segments[2] === "key") {
      const bucket = decodeURIComponent(segments[1]);
      const key = decodeURIComponent(segments[3]);
      if (request.method === "PUT") {
        const value = await parseValue(request);
        const result = await helpers.ObjectDB.put(bucket, key, value);
        return json({ ok: true, bucket, key, result });
      }
      if (request.method === "GET") {
        const value = await helpers.ObjectDB.get(bucket, key);
        return json({ ok: true, bucket, key, value });
      }
      if (request.method === "DELETE") {
        const result = await helpers.ObjectDB.delete(bucket, key);
        return json({ ok: true, bucket, key, result });
      }
      return notFound();
    }

    return notFound();
  },
};
