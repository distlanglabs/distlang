import { simpleApp } from "distlang/layers";
import { helpers } from "distlang";

const configBucket = "simpleapp";
const echoCountKey = "global-echo-count";
const metrics = helpers.instantiateMetrics(
  {
    echoConfigReqs: {
      kind: "counter",
      description: "Number of config requests handled",
      unit: "requests",
      labels: ["route", "method"],
    },
    edgeReqCount: {
      kind: "counter",
      description: "Number of edge requests handled",
      unit: "requests",
      labels: ["route", "method", "status"],
    },
    dbCallLatency: {
      kind: "histogram",
      description: "Latency of ObjectDB calls",
      unit: "ms",
      labels: ["operation"],
    },
  },
  "simpleapp-metrics",
);


const handlerSet1 = {
  routes: {
    POST: {
      "/echo/config": async ({ req, db }) => {
        metrics.echoConfigReqs.inc({ route: "/echo/config", method: "POST" });
        const body = await req.json();
        const requested = Number(body && body.times);
        const times = Number.isFinite(requested) && requested > 0 ? Math.floor(requested) : 1;

        const startedAt = Date.now();
        await db.buckets.create(configBucket);
        await db.put(configBucket, echoCountKey, { times });
        metrics.dbCallLatency.observe(Date.now() - startedAt, { operation: "put" });

        return Response.json({
          ok: true,
          configured: { times },
          target: "handlerSet1",
        });
      },
      "/metrics/query": async ({ req }) => {
        const url = new URL(req.url);
        const minutes = Number(url.searchParams.get("minutes") || "10");
        const end = new Date();
        const start = new Date(end.getTime() - minutes * 60 * 1000);
        const series = await helpers.Metrics.queryRange('increase(edgeReqCount{metricSet="simpleapp-metrics"}[5m])', {
          start: start.toISOString(),
          end: end.toISOString(),
          step: "30s",
        });
        const metadata = await helpers.Metrics.metadata({ metric: "edgeReqCount" });
        const labels = await helpers.Metrics.labels();
        return Response.json({ series, metadata, labels });
      },
    },
  },
};

const handlerSet2 = {
  routes: {
    GET: {
      "/echo/:text": async ({ db, params }) => {
        const startedAt = Date.now();
        const configured = await db.get(configBucket, echoCountKey);
        metrics.dbCallLatency.observe(Date.now() - startedAt, { operation: "get" });
        const times = configured && Number.isFinite(Number(configured.times))
          ? Math.max(1, Math.floor(Number(configured.times)))
          : 1;

        const textToEcho = params && typeof params.text === "string" ? params.text : "hello";
        metrics.edgeReqCount.inc({ route: "/echo/:text", method: "GET", status: "200" });
        return new Response(`${Array(times).fill(textToEcho).join("\n")}\n`);
      },
    },
  },
};

export default simpleApp.instantiate(handlerSet1, handlerSet2, helpers.ObjectDB);
