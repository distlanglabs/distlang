import { simpleApp } from "distlang/layers";
import { helpers } from "distlang";

const configBucket = "simpleapp";
const echoCountKey = "global-echo-count";

const handlerSet1 = {
  routes: {
    POST: {
      "/echo/config": async ({ req, db }) => {
        const body = await req.json();
        const requested = Number(body && body.times);
        const times = Number.isFinite(requested) && requested > 0 ? Math.floor(requested) : 1;

        await db.buckets.create(configBucket);
        await db.put(configBucket, echoCountKey, { times });

        return Response.json({
          ok: true,
          configured: { times },
          target: "handlerSet1",
        });
      },
    },
  },
};

const handlerSet2 = {
  routes: {
    GET: {
      "/echo/:text": async ({ db, params }) => {
        const configured = await db.get(configBucket, echoCountKey);
        const times = configured && Number.isFinite(Number(configured.times))
          ? Math.max(1, Math.floor(Number(configured.times)))
          : 1;

        const textToEcho = params && typeof params.text === "string" ? params.text : "hello";
        return new Response(`${Array(times).fill(textToEcho).join("\n")}\n`);
      },
    },
  },
};

export default simpleApp.instantiate(handlerSet1, handlerSet2, helpers.ObjectDB);
