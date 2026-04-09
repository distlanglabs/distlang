import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { startMockHelpersServer } from "./mock_helpers_server.mjs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const root = path.resolve(__dirname, "..", "..");
const exampleDir = path.join(root, "examples", "metricsApp");

function run(cmd, args, cwd) {
  execFileSync(cmd, args, {
    cwd,
    stdio: "inherit",
    env: process.env,
  });
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function loadWorker(target) {
  const workerPath = path.join(exampleDir, "dist", "v8", target, "worker.js");
  const tempDir = await mkdtemp(path.join(os.tmpdir(), "distlang-metrics-smoke-"));
  const tempPath = path.join(tempDir, "worker.mjs");
  const source = await readFile(workerPath, "utf8");
  await writeFile(tempPath, source, "utf8");
  const url = new URL(`${pathToFileURL(tempPath).href}?t=${Date.now()}`);
  const mod = await import(url.href);
  assert(mod && mod.default && typeof mod.default.fetch === "function", "worker module missing default.fetch");
  return mod.default;
}

async function call(worker, req, env) {
  const request = new Request(`http://example.test${req.path}`, {
    method: req.method || "GET",
    headers: req.headers || {},
    body: req.body,
  });

  const pending = [];
  const response = await worker.fetch(request, env, {
    waitUntil(promise) {
      pending.push(Promise.resolve(promise));
    },
  });
  await Promise.all(pending);

  const text = await response.text();
  let body = text;
  if (text !== "") {
    try {
      body = JSON.parse(text);
    } catch (_err) {
      body = text;
    }
  }

  return { status: response.status, body };
}

async function runMockModeScenario(handlerSet1Worker, handlerSet2Worker) {
  const env = {
    DISTLANG_HELPERS_MODE: "mock",
    __DISTLANG_SIMPLEAPP_TARGET__: "handlerSet1",
  };

  const configRes = await call(handlerSet1Worker, {
    path: "/echo/config",
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ times: 3 }),
  }, env);
  assert(configRes.status === 200, `mock config failed: ${configRes.status}`);
  assert(configRes.body?.configured?.times === 3, `unexpected mock config body: ${JSON.stringify(configRes.body)}`);

  const echoRes = await call(handlerSet2Worker, {
    path: "/echo/hi",
    method: "GET",
  }, {
    DISTLANG_HELPERS_MODE: "mock",
    __DISTLANG_SIMPLEAPP_TARGET__: "handlerSet2",
  });
  assert(echoRes.status === 200, `mock echo failed: ${echoRes.status}`);

  const metricsRes = await call(handlerSet1Worker, {
    path: "/metrics/query",
    method: "POST",
  }, env);
  assert(metricsRes.status === 200, `mock metrics query failed: ${metricsRes.status}`);
  assert(metricsRes.body?.series?.status === "success", `unexpected mock metrics query body: ${JSON.stringify(metricsRes.body)}`);
}

async function runLiveModeScenario(handlerSet1Worker, handlerSet2Worker) {
  const mockServer = await startMockHelpersServer({ token: "metrics-live-token" });
  try {
    const env = {
      DISTLANG_HELPERS_MODE: "live",
      DISTLANG_STORE_BASE_URL: mockServer.baseURL,
      DISTLANG_SERVICE_TOKEN: mockServer.token,
      __DISTLANG_SIMPLEAPP_TARGET__: "handlerSet1",
    };

    const configRes = await call(handlerSet1Worker, {
      path: "/echo/config",
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ times: 4 }),
    }, env);
    assert(configRes.status === 200, `live config failed: ${configRes.status}`);

    const echoRes = await call(handlerSet2Worker, {
      path: "/echo/live",
      method: "GET",
    }, {
      ...env,
      __DISTLANG_SIMPLEAPP_TARGET__: "handlerSet2",
    });
    assert(echoRes.status === 200, `live echo failed: ${echoRes.status}`);

    const metricsReadRes = await call(handlerSet1Worker, {
      path: "/metrics/query",
      method: "POST",
    }, env);
    assert(metricsReadRes.status === 200, `live metrics query failed: ${metricsReadRes.status}`);
    assert(metricsReadRes.body?.series?.status === "success", `unexpected live metrics query body: ${JSON.stringify(metricsReadRes.body)}`);
    assert(metricsReadRes.body?.metadata?.status === "success", `expected metrics metadata response: ${JSON.stringify(metricsReadRes.body)}`);
    assert(Array.isArray(metricsReadRes.body?.labels?.data), `expected metrics labels response: ${JSON.stringify(metricsReadRes.body)}`);

    const metricsCalls = mockServer.calls.filter((entry) => entry.path.startsWith("/metrics/v1/"));
    assert(metricsCalls.length >= 4, `expected metrics helper calls, got ${JSON.stringify(metricsCalls)}`);
    assert(metricsCalls.some((entry) => entry.method === "POST" && entry.path === "/metrics/v1/metricsets/simpleapp-metrics/rows"), "expected metrics row write call");
    assert(metricsCalls.some((entry) => entry.method === "PUT" && entry.path === "/metrics/v1/metricsets/simpleapp-metrics/metadata"), `expected metrics metadata write call, got ${JSON.stringify(metricsCalls)}`);

    const rows = mockServer.analyticsRows.get("simpleapp-metrics") || [];
    assert(rows.length > 0, "expected metrics rows to be written");
    const writtenMetrics = rows.map((row) => row.data?.metric).filter(Boolean);
    assert(writtenMetrics.includes("echoConfigReqs"), `expected echoConfigReqs metric, got ${JSON.stringify(writtenMetrics)}`);
    assert(writtenMetrics.includes("dbCallLatency"), `expected dbCallLatency metric, got ${JSON.stringify(writtenMetrics)}`);
    assert(writtenMetrics.includes("edgeReqCount"), `expected edgeReqCount metric, got ${JSON.stringify(writtenMetrics)}`);
    assert(rows.some((row) => row.data?.labels?.route === "/echo/:text"), `expected labeled metrics rows, got ${JSON.stringify(rows)}`);
  } finally {
    await mockServer.close();
  }
}

async function main() {
  console.log("Building distlang CLI...");
  run("go", ["build", "-o", path.join(root, "bin", "distlang"), "./cmd/distlang"], root);

  console.log("Building metricsApp example...");
  run(path.join(root, "bin", "distlang"), ["build", "index.js"], exampleDir);

  const handlerSet1Worker = await loadWorker("handlerSet1");
  const handlerSet2Worker = await loadWorker("handlerSet2");

  console.log("Running metrics mock mode scenario...");
  await runMockModeScenario(handlerSet1Worker, handlerSet2Worker);

  const liveHandlerSet1Worker = await loadWorker("handlerSet1");
  const liveHandlerSet2Worker = await loadWorker("handlerSet2");

  console.log("Running metrics live mode scenario against local mock helpers server...");
  await runLiveModeScenario(liveHandlerSet1Worker, liveHandlerSet2Worker);

  console.log("helpers_metrics_smoke: PASS");
}

main().catch((err) => {
  console.error("helpers_metrics_smoke: FAIL");
  console.error(err);
  process.exit(1);
});
