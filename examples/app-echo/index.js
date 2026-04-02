import { helpers } from "distlang";
import { app } from "distlang/app";

const configBucket = "app-echo";
const echoCountKey = "global-echo-count";

const appHandlers = {
  routes: {
    POST: {
      "/echo/config": async ({ req, state, params }) => {
        void params;
        const db = state.dbs.ObjectDB;
        const body = await req.json();
        const requested = Number(body && body.times);
        const times = Number.isFinite(requested) && requested > 0 ? Math.floor(requested) : 1;

        await db.buckets.create(configBucket);
        await db.put(configBucket, echoCountKey, { times });

        return {
          ok: true,
          configured: { times },
        };
      },
    },
    GET: {
      "/echo/:text": async ({ req, state, params }) => {
        void req;
        const db = state.dbs.ObjectDB;
        const configured = await db.get(configBucket, echoCountKey);
        const times = configured && Number.isFinite(Number(configured.times))
          ? Math.max(1, Math.floor(Number(configured.times)))
          : 1;

        const textToEcho = typeof params.text === "string" ? params.text : "hello";
        return new Response(`${Array(times).fill(textToEcho).join("\n")}\n`);
      },
    },
  },
};

export default app({
  state: {
    dbs: {
      ObjectDB: helpers.ObjectDB,
    },
  },
  compute: {
    handlers: appHandlers,
  },
});
