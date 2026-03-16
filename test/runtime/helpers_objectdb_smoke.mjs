import { execFileSync } from "node:child_process";
import { mkdtemp, readFile, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { startMockHelpersServer } from "./mock_helpers_server.mjs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const root = path.resolve(__dirname, "..", "..");
const exampleDir = path.join(root, "examples", "helpers-objectdb");

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

async function loadWorker() {
  const workerPath = path.join(exampleDir, "dist", "v8", "worker.js");
  const tempDir = await mkdtemp(path.join(os.tmpdir(), "distlang-smoke-"));
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

  const response = await worker.fetch(request, env, {});
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

async function runMockModeScenario(worker) {
  const env = { DISTLANG_HELPERS_MODE: "mock" };

  const statusRes = await call(worker, { path: "/", method: "GET" }, env);
  assert(statusRes.status === 200, `mock status endpoint failed: ${statusRes.status}`);
  assert(statusRes.body?.status?.mode === "mock", "expected mock mode status response");

  const createRes = await call(worker, { path: "/bucket/demo", method: "PUT" }, env);
  assert(createRes.status === 200, `mock create bucket failed: ${createRes.status}`);

  const putRes = await call(
    worker,
    {
      path: "/bucket/demo/key/hello",
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message: "hi-mock" }),
    },
    env,
  );
  assert(putRes.status === 200, `mock put failed: ${putRes.status}`);

  const getRes = await call(worker, { path: "/bucket/demo/key/hello", method: "GET" }, env);
  assert(getRes.status === 200, `mock get failed: ${getRes.status}`);
  assert(getRes.body?.value?.message === "hi-mock", `mock get unexpected body: ${JSON.stringify(getRes.body)}`);
}

async function runLiveModeScenario(worker) {
  const mockServer = await startMockHelpersServer({ token: "live-test-token" });
  try {
    const env = {
      DISTLANG_HELPERS_MODE: "live",
      DISTLANG_STORE_BASE_URL: mockServer.baseURL,
      DISTLANG_SERVICE_TOKEN: mockServer.token,
    };

    const statusRes = await call(worker, { path: "/", method: "GET" }, env);
    assert(statusRes.status === 200, `live status endpoint failed: ${statusRes.status}`);
    assert(statusRes.body?.status?.service === "objectdb", `unexpected live status body: ${JSON.stringify(statusRes.body)}`);

    await call(worker, { path: "/bucket/demo-live", method: "PUT" }, env);

    const putRes = await call(
      worker,
      {
        path: "/bucket/demo-live/key/hello",
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: "hi-live" }),
      },
      env,
    );
    assert(putRes.status === 200, `live put failed: ${putRes.status}`);

    const getRes = await call(worker, { path: "/bucket/demo-live/key/hello", method: "GET" }, env);
    assert(getRes.status === 200, `live get failed: ${getRes.status}`);
    assert(getRes.body?.value?.message === "hi-live", `live get unexpected body: ${JSON.stringify(getRes.body)}`);

    assert(mockServer.calls.length > 0, "expected live scenario to call mock helpers server");
  } finally {
    await mockServer.close();
  }
}

async function runMissingTokenScenario(worker) {
  const env = {
    DISTLANG_HELPERS_MODE: "live",
    DISTLANG_STORE_BASE_URL: "http://127.0.0.1:1",
  };

  let error = null;
  try {
    await call(worker, { path: "/", method: "GET" }, env);
  } catch (err) {
    error = err;
  }
  assert(error, "expected live mode without token to throw");
  assert(String(error.message || error).includes("DISTLANG_SERVICE_TOKEN"), `unexpected missing token error: ${error}`);
}

async function main() {
  console.log("Building distlang CLI...");
  run("go", ["build", "-o", path.join(root, "bin", "distlang"), "./cmd/distlang"], root);

  console.log("Building helpers-objectdb example...");
  run(path.join(root, "bin", "distlang"), ["build", "index.js"], exampleDir);

  const worker = await loadWorker();

  console.log("Running mock mode scenario...");
  await runMockModeScenario(worker);

  console.log("Running live mode scenario against local mock helpers server...");
  await runLiveModeScenario(worker);

  console.log("Running missing token scenario...");
  await runMissingTokenScenario(worker);

  console.log("helpers_objectdb_smoke: PASS");
}

main().catch((err) => {
  console.error("helpers_objectdb_smoke: FAIL");
  console.error(err);
  process.exit(1);
});
