import { simpleApp } from "distlang/layers";
import { helpers } from "distlang";

const configBucket = "simpleapp";
const echoCountKey = "global-echo-count";
const metrics = helpers.instantiateMetrics(
  {
    echoConfigReqs: "counter",
    edgeReqCount: "counter",
    dbCallLatency: "histogram",
  },
  "simpleApp-metrics",
);


const handlerSet1 = {
  routes: {
    POST: {
      "/echo/config": async ({ req, db }) => {
        metrics.echoConfigReqs.inc();
        const body = await req.json();
        const requested = Number(body && body.times);
        const times = Number.isFinite(requested) && requested > 0 ? Math.floor(requested) : 1;

        const startedAt = Date.now();
        await db.buckets.create(configBucket);
        await db.put(configBucket, echoCountKey, { times });
        metrics.dbCallLatency.observe(Date.now() - startedAt);

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
        metrics.edgeReqCount.inc();
        const startedAt = Date.now();
        const configured = await db.get(configBucket, echoCountKey);
        metrics.dbCallLatency.observe(Date.now() - startedAt);
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
