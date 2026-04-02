import { helpersObjectDB } from "./objectdb.js";
import { helpersMetrics, instantiateMetrics, queueFlushAllMetricsStates, settleAllMetricFlushes } from "./metrics.js";
import { clearRuntimeContext, runtimeState, setRuntimeContext } from "./shared.js";

export function wrapWorkerWithHelpers(worker) {
  return {
    ...worker,
    async fetch(request, env, ctx) {
      setRuntimeContext(env, ctx);
      try {
        return await worker.fetch(request, env, ctx);
      } finally {
        queueFlushAllMetricsStates();
        if (!runtimeState.currentCtx || typeof runtimeState.currentCtx.waitUntil !== "function") {
          await settleAllMetricFlushes();
        }
        clearRuntimeContext();
      }
    },
  };
}

export const helpers = {
  ObjectDB: helpersObjectDB,
  Metrics: helpersMetrics,
  instantiateMetrics,
};
