package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	localmetrics "github.com/distlanglabs/distlang/pkg/local/metrics"
)

type Config struct {
	Host  string
	Port  int
	Store localmetrics.Store
}

type Running struct {
	server *http.Server
	ln     net.Listener
	store  localmetrics.Store
}

func Start(cfg Config) (*Running, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 4817
	}
	store := cfg.Store
	if store == nil {
		store = localmetrics.NewMemoryStore()
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	r := &Running{ln: ln, store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("/", r.handle)
	r.server = &http.Server{Handler: mux}
	go func() { _ = r.server.Serve(ln) }()
	return r, nil
}

func (r *Running) URL() string {
	return "http://" + r.ln.Addr().String()
}

func (r *Running) DistlangURL() string {
	return r.URL() + "/distlang"
}

func (r *Running) Close(ctx context.Context) error {
	serverErr := r.server.Shutdown(ctx)
	storeErr := r.store.Close()
	if serverErr != nil {
		return serverErr
	}
	return storeErr
}

func (r *Running) handle(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/distlang")
	if path == "" {
		path = "/"
	}

	if req.Method == http.MethodGet && path == "/" {
		writeHTML(w, http.StatusOK, localHomeHTML())
		return
	}
	if req.Method == http.MethodGet && path == "/metrics" {
		writeHTML(w, http.StatusOK, localMetricsHTML())
		return
	}
	if req.Method == http.MethodGet && path == "/metrics/v1/api/v1/metadata" {
		r.handleMetadata(w, req)
		return
	}
	if req.Method == http.MethodGet && path == "/metrics/v1/api/v1/query" {
		r.handleQuery(w, req)
		return
	}
	if req.Method == http.MethodGet && path == "/metrics/v1/api/v1/query_range" {
		r.handleQueryRange(w, req)
		return
	}
	if strings.HasPrefix(path, "/metrics/v1/metricsets/") {
		r.handleMetricSet(w, req, path)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "route not found"})
}

func (r *Running) handleMetricSet(w http.ResponseWriter, req *http.Request, path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/metrics/v1/metricsets/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_metric_set", "message": "metric set is required"})
		return
	}
	metricSet, _ := url.PathUnescape(parts[0])

	if len(parts) == 1 && req.Method == http.MethodPut {
		if err := r.store.EnsureMetricSet(req.Context(), metricSet, nil); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "metricSet": metricSet})
		return
	}
	if len(parts) == 2 && parts[1] == "metadata" && req.Method == http.MethodPut {
		var payload struct {
			Metrics map[string]localmetrics.Definition `json:"metrics"`
		}
		if err := readJSON(req, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_body", "message": err.Error()})
			return
		}
		if err := r.store.EnsureMetricSet(req.Context(), metricSet, payload.Metrics); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "metricSet": metricSet})
		return
	}
	if len(parts) == 2 && parts[1] == "rows" && req.Method == http.MethodPost {
		var payload struct {
			Rows []struct {
				TS   string           `json:"ts"`
				Data localmetrics.Row `json:"data"`
			} `json:"rows"`
		}
		if err := readJSON(req, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_body", "message": err.Error()})
			return
		}
		rows := make([]localmetrics.Row, 0, len(payload.Rows))
		for _, item := range payload.Rows {
			row := item.Data
			if row.WindowStart.IsZero() && strings.TrimSpace(item.TS) != "" {
				parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(item.TS))
				if err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_ts", "message": err.Error()})
					return
				}
				row.WindowStart = parsed
			}
			rows = append(rows, row)
		}
		if err := r.store.AppendRows(req.Context(), metricSet, rows); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "metricSet": metricSet, "rows": len(rows)})
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "route not found"})
}

func (r *Running) handleMetadata(w http.ResponseWriter, req *http.Request) {
	metadata, err := r.store.Metadata(req.Context(), req.URL.Query().Get("metric"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": metadata})
}

func (r *Running) handleQuery(w http.ResponseWriter, req *http.Request) {
	parsed, err := parseQuery(req.URL.Query().Get("query"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	evalTime := time.Now().UTC()
	if raw := strings.TrimSpace(req.URL.Query().Get("time")); raw != "" {
		if parsedTime, err := parsePromTime(raw); err == nil {
			evalTime = parsedTime
		}
	}
	value, err := r.evaluate(req.Context(), parsed, evalTime)
	if err != nil {
		writeError(w, err)
		return
	}
	result := []any{}
	if value != 0 {
		result = append(result, map[string]any{
			"metric": map[string]string{"__name__": parsed.Metric, "metricSet": parsed.MetricSet},
			"value":  []any{float64(evalTime.UnixNano()) / 1e9, strconv.FormatFloat(value, 'f', -1, 64)},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"resultType": "vector", "result": result}})
}

func (r *Running) handleQueryRange(w http.ResponseWriter, req *http.Request) {
	parsed, err := parseQuery(req.URL.Query().Get("query"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	start, err := parsePromTime(req.URL.Query().Get("start"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": "invalid start"})
		return
	}
	end, err := parsePromTime(req.URL.Query().Get("end"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": "invalid end"})
		return
	}
	step, err := parsePromDuration(req.URL.Query().Get("step"))
	if err != nil || step <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": "invalid step"})
		return
	}
	values := []any{}
	for ts := start; !ts.After(end); ts = ts.Add(step) {
		value, err := r.evaluate(req.Context(), parsed, ts)
		if err != nil {
			writeError(w, err)
			return
		}
		values = append(values, []any{float64(ts.UnixNano()) / 1e9, strconv.FormatFloat(value, 'f', -1, 64)})
	}
	result := []any{}
	if len(values) > 0 {
		result = append(result, map[string]any{"metric": map[string]string{"__name__": parsed.Metric, "metricSet": parsed.MetricSet}, "values": values})
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"resultType": "matrix", "result": result}})
}

type promQuery struct {
	Function  string
	Metric    string
	MetricSet string
	Range     time.Duration
}

var increaseRE = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\(([a-zA-Z_:][a-zA-Z0-9_:]*)\{metricSet="([^"]+)"\}\[([^\]]+)\]\)$`)

func parseQuery(raw string) (promQuery, error) {
	match := increaseRE.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return promQuery{}, fmt.Errorf("unsupported local metrics query: %s", raw)
	}
	if match[1] != "increase" && match[1] != "sum_over_time" {
		return promQuery{}, fmt.Errorf("unsupported local metrics function: %s", match[1])
	}
	duration, err := parsePromDuration(match[4])
	if err != nil {
		return promQuery{}, err
	}
	return promQuery{Function: match[1], Metric: match[2], MetricSet: match[3], Range: duration}, nil
}

func (r *Running) evaluate(ctx context.Context, query promQuery, evalTime time.Time) (float64, error) {
	rows, err := r.store.LoadRows(ctx, localmetrics.RowQuery{MetricSet: query.MetricSet, Metric: query.Metric, Start: evalTime.Add(-query.Range), End: evalTime})
	if err != nil {
		return 0, err
	}
	var total float64
	for _, row := range rows {
		if row.Row.Kind == "histogram" && len(row.Row.Values) > 0 {
			for _, value := range row.Row.Values {
				total += value
			}
			continue
		}
		total += row.Row.Sum
	}
	return total, nil
}

func parsePromDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("duration is required")
	}
	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(raw)
}

func parsePromTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("time is required")
	}
	if numeric, err := strconv.ParseFloat(raw, 64); err == nil {
		sec, frac := mathModf(numeric)
		return time.Unix(int64(sec), int64(frac*1e9)).UTC(), nil
	}
	return time.Parse(time.RFC3339Nano, raw)
}

func mathModf(v float64) (float64, float64) {
	whole := float64(int64(v))
	return whole, v - whole
}

func readJSON(req *http.Request, out any) error {
	body, err := io.ReadAll(io.LimitReader(req.Body, 10<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, localmetrics.ErrInvalidMetricSet) {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]any{"error": "local_metrics_error", "message": err.Error()})
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func localHomeHTML() string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Distlang Local</title>
  <style>
    body { margin: 0; font-family: ui-sans-serif, system-ui, sans-serif; background: #0f172a; color: #e2e8f0; }
    main { max-width: 760px; margin: 0 auto; padding: 64px 24px; }
    a { color: #67e8f9; }
    .card { margin-top: 24px; padding: 24px; border: 1px solid #334155; border-radius: 16px; background: #111827; }
  </style>
</head>
<body>
  <main>
    <h1>Distlang Local</h1>
    <p>Local Distlang is running with in-memory Metrics storage.</p>
    <div class="card">
      <h2>Metrics</h2>
      <p>Inspect metric definitions and run local Metrics queries.</p>
      <p><a href="/distlang/metrics">Open Local Metrics Explorer</a></p>
    </div>
  </main>
</body>
</html>`
}

func localMetricsHTML() string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Local Metrics Explorer</title>
  <style>
    :root { color-scheme: dark; }
    body { margin: 0; font-family: ui-sans-serif, system-ui, sans-serif; background: #07111f; color: #dbeafe; }
    main { max-width: 1120px; margin: 0 auto; padding: 32px 20px 56px; }
    header { display: flex; align-items: end; justify-content: space-between; gap: 16px; margin-bottom: 24px; }
    h1 { margin: 0; font-size: clamp(28px, 6vw, 48px); letter-spacing: -0.04em; }
    p { color: #93a4b8; }
    button, input { border: 1px solid #334155; border-radius: 10px; background: #0f172a; color: #e2e8f0; padding: 10px 12px; font: inherit; }
    button { cursor: pointer; background: #155e75; border-color: #0891b2; }
    button:hover { background: #0e7490; }
    .grid { display: grid; grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr); gap: 18px; }
    .card { border: 1px solid #1e3a5f; border-radius: 18px; background: linear-gradient(180deg, #0f172a, #0b1220); padding: 18px; box-shadow: 0 24px 80px rgb(0 0 0 / 0.24); }
    .toolbar { display: flex; gap: 8px; margin: 12px 0 16px; }
    .toolbar input { flex: 1; min-width: 0; }
    .metric { width: 100%; text-align: left; margin: 8px 0; background: #111827; border-color: #24364f; }
    .metric small { display: block; color: #93a4b8; margin-top: 4px; }
    pre { overflow: auto; min-height: 220px; margin: 0; padding: 16px; border-radius: 14px; background: #020617; color: #bfdbfe; }
    .status { min-height: 20px; margin: 8px 0 0; color: #67e8f9; }
    @media (max-width: 800px) { header, .grid { display: block; } .card { margin-bottom: 18px; } }
  </style>
</head>
<body>
  <main>
    <header>
      <div>
        <h1>Local Metrics Explorer</h1>
        <p>Reads from the Metrics store running inside this <code>distlang local</code> process.</p>
      </div>
      <button id="refresh">Refresh Metadata</button>
    </header>
    <section class="grid">
      <div class="card">
        <h2>Metric Definitions</h2>
        <p>Select a metric to build an instant query.</p>
        <div id="metrics"></div>
      </div>
      <div class="card">
        <h2>Query</h2>
        <div class="toolbar">
          <input id="query" spellcheck="false" placeholder='increase(trafficReqCount{metricSet="local-metrics-tests-app"}[5m])'>
          <button id="run">Run</button>
        </div>
        <p class="status" id="status"></p>
        <pre id="output">Run a query to inspect local Metrics data.</pre>
      </div>
    </section>
  </main>
  <script>
    const metricsEl = document.querySelector("#metrics");
    const queryEl = document.querySelector("#query");
    const outputEl = document.querySelector("#output");
    const statusEl = document.querySelector("#status");

    function renderJSON(value) {
      outputEl.textContent = JSON.stringify(value, null, 2);
    }

    async function loadMetadata() {
      statusEl.textContent = "Loading metadata...";
      const response = await fetch("/distlang/metrics/v1/api/v1/metadata");
      const payload = await response.json();
      const data = payload.data || {};
      const entries = Object.entries(data).flatMap(([name, values]) => values.map((entry) => ({ name, ...entry })));
      metricsEl.innerHTML = "";
      if (entries.length === 0) {
        metricsEl.innerHTML = "<p>No local metrics have been recorded yet.</p>";
        statusEl.textContent = "No metadata yet.";
        return;
      }
      for (const entry of entries) {
        const button = document.createElement("button");
        button.className = "metric";
        button.innerHTML = "<strong>" + entry.name + "</strong><small>" + (entry.type || "metric") + " - " + (entry.metricSet || "unknown metric set") + "</small>";
        button.addEventListener("click", () => {
          queryEl.value = "increase(" + entry.name + "{metricSet=\"" + entry.metricSet + "\"}[5m])";
        });
        metricsEl.appendChild(button);
      }
      statusEl.textContent = entries.length + " metric definition" + (entries.length === 1 ? "" : "s") + " loaded.";
    }

    async function runQuery() {
      const query = queryEl.value.trim();
      if (!query) {
        statusEl.textContent = "Enter a query first.";
        return;
      }
      statusEl.textContent = "Running query...";
      const response = await fetch("/distlang/metrics/v1/api/v1/query?query=" + encodeURIComponent(query));
      const payload = await response.json();
      renderJSON(payload);
      statusEl.textContent = response.ok ? "Query complete." : "Query failed.";
    }

    document.querySelector("#refresh").addEventListener("click", loadMetadata);
    document.querySelector("#run").addEventListener("click", runQuery);
    queryEl.addEventListener("keydown", (event) => {
      if (event.key === "Enter") runQuery();
    });
    loadMetadata().catch((error) => {
      statusEl.textContent = "Metadata request failed.";
      outputEl.textContent = error instanceof Error ? error.message : String(error);
    });
  </script>
</body>
</html>`
}
