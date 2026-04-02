import { helpers } from "distlang";
import { app } from "distlang/app";

const configBucket = "app-echo";
const echoCountKey = "global-echo-count";
const appMetrics = helpers.instantiateMetrics(
  {
    echoConfigReqs: {
      kind: "counter",
      description: "Number of config requests handled",
      unit: "requests",
      labels: ["route", "method"],
    },
    echoReqCount: {
      kind: "counter",
      description: "Number of echo requests handled",
      unit: "requests",
      labels: ["route", "method", "status"],
    },
  },
  "app-echo-metrics",
);

const appHandlers = {
  routes: {
    POST: {
      "/echo/config": async ({ req, state, params }) => {
        void params;
        const db = state.dbs.ObjectDB;
        const metrics = state.observability.AppMetrics;
        metrics.echoConfigReqs.inc({ route: "/echo/config", method: "POST" });
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
        const metrics = state.observability.AppMetrics;
        const configured = await db.get(configBucket, echoCountKey);
        const times = configured && Number.isFinite(Number(configured.times))
          ? Math.max(1, Math.floor(Number(configured.times)))
          : 1;

        const textToEcho = typeof params.text === "string" ? params.text : "hello";
        metrics.echoReqCount.inc({ route: "/echo/:text", method: "GET", status: "200" });
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
    observability: {
      AppMetrics: appMetrics,
    },
  },
  compute: {
    handlers: appHandlers,
  },
});
