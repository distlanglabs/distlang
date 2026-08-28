package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/distlanglabs/distlang/pkg/auth"
	localmetrics "github.com/distlanglabs/distlang/pkg/local/metrics"
	storeapi "github.com/distlanglabs/distlang/pkg/store"
)

type Config struct {
	Host              string
	Port              int
	Store             localmetrics.Store
	Backend           localmetrics.QueryBackend
	HostedBaseURL     string
	HostedAccessToken string
	HostedToken       func() (string, error)
	HTTPClient        *http.Client
}

type Running struct {
	server  *http.Server
	ln      net.Listener
	store   localmetrics.Store
	backend localmetrics.QueryBackend

	hostedBaseURL     string
	hostedAccessToken string
	hostedToken       func() (string, error)
	httpClient        *http.Client
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
	backend := cfg.Backend
	if backend == nil {
		backend = localmetrics.NewMemoryQueryBackend(store)
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	r := &Running{ln: ln, store: store, backend: backend, hostedBaseURL: strings.TrimRight(strings.TrimSpace(cfg.HostedBaseURL), "/"), hostedAccessToken: strings.TrimSpace(cfg.HostedAccessToken), hostedToken: cfg.HostedToken, httpClient: cfg.HTTPClient}
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

func (r *Running) queryBackend() localmetrics.QueryBackend {
	if r.backend != nil {
		return r.backend
	}
	return localmetrics.NewMemoryQueryBackend(r.store)
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
	if req.Method == http.MethodGet && path == "/metrics/v1/capabilities" {
		r.handleCapabilities(w, req)
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
	if req.Method == http.MethodPost && path == "/metrics/v1/sql" {
		r.handleSQL(w, req)
		return
	}
	if req.Method == http.MethodGet && path == "/metrics/hosted/v1/status" {
		r.handleHostedStatus(w, req)
		return
	}
	if strings.HasPrefix(path, "/metrics/hosted/v1/") {
		r.handleHostedProxy(w, req, path)
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
	metadata, err := r.queryBackend().Metadata(req.Context(), localmetrics.MetadataRequest{Metric: req.URL.Query().Get("metric")})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, metadata)
}

func (r *Running) handleCapabilities(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusOK, r.queryBackend().Capabilities(req.Context()))
}

func (r *Running) handleQuery(w http.ResponseWriter, req *http.Request) {
	evalTime := time.Now().UTC()
	if raw := strings.TrimSpace(req.URL.Query().Get("time")); raw != "" {
		if parsedTime, err := localmetrics.ParsePromTime(raw); err == nil {
			evalTime = parsedTime
		}
	}
	result, err := r.queryBackend().Query(req.Context(), localmetrics.QueryRequest{Query: req.URL.Query().Get("query"), Time: evalTime})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (r *Running) handleQueryRange(w http.ResponseWriter, req *http.Request) {
	start, err := localmetrics.ParsePromTime(req.URL.Query().Get("start"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": "invalid start"})
		return
	}
	end, err := localmetrics.ParsePromTime(req.URL.Query().Get("end"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": "invalid end"})
		return
	}
	step, err := localmetrics.ParsePromDuration(req.URL.Query().Get("step"))
	if err != nil || step <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": "invalid step"})
		return
	}
	result, err := r.queryBackend().QueryRange(req.Context(), localmetrics.QueryRangeRequest{Query: req.URL.Query().Get("query"), Start: start, End: end, Step: step})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (r *Running) handleSQL(w http.ResponseWriter, req *http.Request) {
	var sqlReq localmetrics.SQLQueryRequest
	if err := readJSON(req, &sqlReq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_body", "message": err.Error()})
		return
	}
	result, err := r.queryBackend().SQL(req.Context(), sqlReq)
	if err != nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "sql_unavailable", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (r *Running) handleHostedStatus(w http.ResponseWriter, req *http.Request) {
	if r.hostedAccessToken != "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"loggedIn":     true,
			"authBaseURL":  auth.ResolveBaseURL(),
			"storeBaseURL": r.hostedMetricsBaseURL(),
		})
		return
	}
	session, err := auth.LoadSession()
	if err != nil {
		status := map[string]any{
			"ok":           true,
			"loggedIn":     false,
			"message":      "Run `distlang helpers login` to query hosted Metrics from this explorer.",
			"authBaseURL":  auth.ResolveBaseURL(),
			"storeBaseURL": storeapi.ResolveBaseURL(),
		}
		if !errors.Is(err, auth.ErrNotLoggedIn) {
			status["ok"] = false
			status["error"] = "auth_status_failed"
			status["message"] = err.Error()
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	user := map[string]string{"id": session.User.ID, "email": session.User.Email, "name": session.User.Name}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"loggedIn":     strings.TrimSpace(session.AccessToken) != "",
		"user":         user,
		"authBaseURL":  auth.ResolveBaseURL(),
		"storeBaseURL": storeapi.ResolveBaseURL(),
	})
}

func (r *Running) handleHostedProxy(w http.ResponseWriter, req *http.Request, path string) {
	token, err := r.hostedBearerToken()
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "hosted_not_logged_in", "message": "Run `distlang helpers login` to query hosted Metrics from this explorer."})
		return
	}
	remoteURL := r.hostedMetricsBaseURL() + "/distlang/metrics/v1" + strings.TrimPrefix(path, "/metrics/hosted/v1")
	var body io.Reader
	if req.Body != nil {
		body = req.Body
	}
	proxyReq, err := http.NewRequestWithContext(req.Context(), req.Method, remoteURL, body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "hosted_proxy_failed", "message": err.Error()})
		return
	}
	proxyReq.Header.Set("Authorization", "Bearer "+token)
	if contentType := req.Header.Get("Content-Type"); contentType != "" {
		proxyReq.Header.Set("Content-Type", contentType)
	}
	res, err := r.httpClientForProxy().Do(proxyReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "hosted_proxy_failed", "message": err.Error()})
		return
	}
	defer res.Body.Close()
	for key, values := range res.Header {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(res.Body, 4<<20))
}

func (r *Running) hostedMetricsBaseURL() string {
	if r.hostedBaseURL != "" {
		return r.hostedBaseURL
	}
	return storeapi.ResolveBaseURL()
}

func (r *Running) hostedBearerToken() (string, error) {
	if r.hostedAccessToken != "" {
		return r.hostedAccessToken, nil
	}
	if r.hostedToken != nil {
		return r.hostedToken()
	}
	session, err := auth.NewClient(auth.ResolveBaseURL()).EnsureSession()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(session.AccessToken) == "" {
		return "", auth.ErrNotLoggedIn
	}
	return session.AccessToken, nil
}

func (r *Running) httpClientForProxy() *http.Client {
	if r.httpClient != nil {
		return r.httpClient
	}
	return &http.Client{Timeout: 30 * time.Second}
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
    button, input, textarea { border: 1px solid #334155; border-radius: 10px; background: #0f172a; color: #e2e8f0; padding: 10px 12px; font: inherit; }
    textarea { box-sizing: border-box; min-height: 90px; resize: vertical; width: 100%; }
    button { cursor: pointer; background: #155e75; border-color: #0891b2; }
    button:hover { background: #0e7490; }
    .grid { display: grid; grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr); gap: 18px; }
    .card { border: 1px solid #1e3a5f; border-radius: 18px; background: linear-gradient(180deg, #0f172a, #0b1220); padding: 18px; box-shadow: 0 24px 80px rgb(0 0 0 / 0.24); }
    .toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0 16px; }
    .toolbar input { flex: 1; min-width: 0; }
    .metric { width: 100%; text-align: left; margin: 8px 0; background: #111827; border-color: #24364f; }
    .metric small { display: block; color: #93a4b8; margin-top: 4px; }
    pre { overflow: auto; min-height: 220px; margin: 0; padding: 16px; border-radius: 14px; background: #020617; color: #bfdbfe; }
    .status { min-height: 20px; margin: 8px 0 0; color: #67e8f9; }
    .pill { display: inline-flex; gap: 6px; align-items: center; margin: 4px 6px 0 0; padding: 5px 9px; border: 1px solid #1e3a5f; border-radius: 999px; background: #0f172a; color: #bae6fd; font-size: 13px; }
    .sourcebar { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-top: 14px; }
    .source { background: #111827; border-color: #24364f; }
    .source.active { background: #155e75; border-color: #0891b2; }
    .disabled { opacity: 0.6; }
    .tabs { display: flex; gap: 8px; margin: 10px 0; }
    .tab { background: #111827; border-color: #24364f; }
    .tab.active { background: #155e75; border-color: #0891b2; }
    .result-pane { display: none; }
    .result-pane.active { display: block; }
    .table-wrap { max-height: 520px; overflow: auto; border: 1px solid #1e3a5f; border-radius: 14px; background: #020617; }
    table { width: 100%; border-collapse: separate; border-spacing: 0; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: 13px; }
    th, td { padding: 9px 11px; border-bottom: 1px solid #13253c; text-align: left; vertical-align: top; white-space: nowrap; }
    th { position: sticky; top: 0; z-index: 1; background: #0f172a; color: #93c5fd; font-weight: 650; }
    td.null { color: #64748b; font-style: italic; }
    .empty { margin: 0; padding: 16px; color: #93a4b8; }
    @media (max-width: 800px) { header, .grid { display: block; } .card { margin-bottom: 18px; } }
  </style>
</head>
<body>
  <main>
    <header>
      <div>
        <h1>Local Metrics Explorer</h1>
        <p>Inspect local Metrics data or switch to hosted account data through this <code>distlang local</code> explorer.</p>
        <div id="capabilities"></div>
        <div class="sourcebar" id="sources"></div>
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
    <section class="card disabled" id="sql-card" style="margin-top: 18px;">
      <h2>SQL Query</h2>
      <p id="sql-status">SQL is loading backend capabilities.</p>
      <textarea id="sql-query" spellcheck="false">select metric_set, metric, kind, window_start, count, sum from metric_rows order by window_start desc limit 20</textarea>
      <div class="toolbar">
        <button id="run-sql" disabled>Run SQL</button>
        <span class="toolbar" id="sql-examples"></span>
      </div>
      <div class="tabs" aria-label="SQL result view">
        <button class="tab active" id="sql-table-tab" type="button">Table</button>
        <button class="tab" id="sql-raw-tab" type="button">Raw JSON</button>
      </div>
      <div class="result-pane active" id="sql-table-pane"><p class="empty">SQL table results will appear here.</p></div>
      <pre class="result-pane" id="sql-output">Raw SQL JSON will appear here.</pre>
      <pre id="schema">Loading schema hints...</pre>
    </section>
  </main>
  <script>
    window.__DISTLANG_EXPLORER_CONFIG__ = {
      defaultSource: "local",
      sources: {
        local: {
          label: "Local SQLite",
          mode: "local",
          auth: false,
          apiBasePath: "/distlang/metrics/v1"
        },
        hosted: {
          label: "Hosted Distlang",
          mode: "hosted",
          auth: true,
          apiBasePath: "/distlang/metrics/hosted/v1",
          statusPath: "/distlang/metrics/hosted/v1/status"
        }
      }
    };
    const explorerConfig = window.__DISTLANG_EXPLORER_CONFIG__;
    let activeSourceName = explorerConfig.defaultSource;
    let activeSource = explorerConfig.sources[activeSourceName];
    const metricsEl = document.querySelector("#metrics");
    const queryEl = document.querySelector("#query");
    const outputEl = document.querySelector("#output");
    const statusEl = document.querySelector("#status");
    const capabilitiesEl = document.querySelector("#capabilities");
    const sourcesEl = document.querySelector("#sources");
    const sqlCardEl = document.querySelector("#sql-card");
    const sqlStatusEl = document.querySelector("#sql-status");
    const sqlQueryEl = document.querySelector("#sql-query");
    const sqlOutputEl = document.querySelector("#sql-output");
    const sqlTablePaneEl = document.querySelector("#sql-table-pane");
    const sqlTableTabEl = document.querySelector("#sql-table-tab");
    const sqlRawTabEl = document.querySelector("#sql-raw-tab");
    const runSQLEl = document.querySelector("#run-sql");
    const sqlExamplesEl = document.querySelector("#sql-examples");
    const schemaEl = document.querySelector("#schema");
    let capabilities = null;

    function apiPath(path) {
      return activeSource.apiBasePath.replace(/\/$/, "") + path;
    }

    function renderSources() {
      sourcesEl.innerHTML = "";
      for (const [name, source] of Object.entries(explorerConfig.sources || {})) {
        const button = document.createElement("button");
        button.className = "source" + (name === activeSourceName ? " active" : "");
        button.type = "button";
        button.textContent = source.label || name;
        button.addEventListener("click", () => switchSource(name));
        sourcesEl.appendChild(button);
      }
    }

    async function switchSource(name) {
      activeSourceName = name;
      activeSource = explorerConfig.sources[name];
      capabilities = null;
      renderSources();
      metricsEl.innerHTML = "";
      outputEl.textContent = "Run a query to inspect " + (activeSource.label || name) + " Metrics data.";
      sqlTablePaneEl.innerHTML = '<p class="empty">SQL table results will appear here.</p>';
      sqlOutputEl.textContent = "Raw SQL JSON will appear here.";
      await Promise.all([loadCapabilities(), loadMetadata()]);
    }

    async function ensureHostedReady() {
      if (activeSource.mode !== "hosted") return true;
      const response = await fetch(activeSource.statusPath);
      const status = await response.json();
      if (status.loggedIn) {
        statusEl.textContent = "Hosted account loaded" + (status.user && status.user.email ? ": " + status.user.email : ".");
        return true;
      }
      capabilities = { mode: "hosted", storage: "durable_object_sqlite", auth: true, features: { metadata: false, query: false, queryRange: false, sql: false }, schema: [] };
      capabilitiesEl.innerHTML = "";
      for (const item of ["mode", "storage", "auth"]) {
        const span = document.createElement("span");
        span.className = "pill";
        span.textContent = item + ": " + capabilities[item];
        capabilitiesEl.appendChild(span);
      }
      metricsEl.innerHTML = "<p>Hosted Metrics requires CLI auth. Run <code>distlang helpers login</code>, then refresh this explorer.</p>";
      statusEl.textContent = status.message || "Hosted Metrics requires login.";
      sqlStatusEl.textContent = "Hosted SQL requires login.";
      runSQLEl.disabled = true;
      renderSQLExamples([], false);
      schemaEl.textContent = "[]";
      return false;
    }

    function renderJSON(value) {
      outputEl.textContent = JSON.stringify(value, null, 2);
    }

    function setSQLView(view) {
      const tableActive = view === "table";
      sqlTableTabEl.classList.toggle("active", tableActive);
      sqlRawTabEl.classList.toggle("active", !tableActive);
      sqlTablePaneEl.classList.toggle("active", tableActive);
      sqlOutputEl.classList.toggle("active", !tableActive);
    }

    function formatSQLCell(value) {
      if (value === null || value === undefined) return "NULL";
      if (typeof value === "object") return JSON.stringify(value);
      return String(value);
    }

    function renderSQLTable(payload) {
      const columns = Array.isArray(payload.columns) ? payload.columns : [];
      const rows = Array.isArray(payload.rows) ? payload.rows : [];
      if (columns.length === 0) {
        sqlTablePaneEl.innerHTML = '<p class="empty">No columns returned.</p>';
        return;
      }
      const wrapper = document.createElement("div");
      wrapper.className = "table-wrap";
      const table = document.createElement("table");
      const thead = document.createElement("thead");
      const headerRow = document.createElement("tr");
      for (const column of columns) {
        const th = document.createElement("th");
        th.textContent = column.name || "column";
        if (column.type) th.title = column.type;
        headerRow.appendChild(th);
      }
      thead.appendChild(headerRow);
      table.appendChild(thead);
      const tbody = document.createElement("tbody");
      for (const row of rows) {
        const tr = document.createElement("tr");
        for (let i = 0; i < columns.length; i++) {
          const td = document.createElement("td");
          const value = Array.isArray(row) ? row[i] : null;
          td.textContent = formatSQLCell(value);
          if (value === null || value === undefined) td.className = "null";
          tr.appendChild(td);
        }
        tbody.appendChild(tr);
      }
      table.appendChild(tbody);
      wrapper.appendChild(table);
      sqlTablePaneEl.innerHTML = "";
      if (rows.length === 0) {
        sqlTablePaneEl.innerHTML = '<p class="empty">Query returned 0 rows.</p>';
        return;
      }
      sqlTablePaneEl.appendChild(wrapper);
    }

    function renderSQLExamples(schema, enabled) {
      const tables = Array.isArray(schema) ? schema : [];
      const tableNames = tables.map((table) => table.name).filter(Boolean);
      const sampleTable = tableNames.includes("metric_rows") ? "metric_rows" : tableNames[0];
      const examples = [{ label: "Show Tables", sql: "show tables;" }];
      if (sampleTable) {
        examples.push({ label: "Describe " + sampleTable, sql: "describe " + sampleTable + ";" });
        examples.push({ label: "Sample Rows", sql: sampleRowsSQL(tables.find((table) => table.name === sampleTable) || { name: sampleTable }) });
      }
      sqlExamplesEl.innerHTML = "";
      for (const example of examples) {
        const button = document.createElement("button");
        button.className = "sql-example";
        button.type = "button";
        button.disabled = !enabled;
        button.textContent = example.label;
        button.addEventListener("click", () => {
          sqlQueryEl.value = example.sql;
          runSQL();
        });
        sqlExamplesEl.appendChild(button);
      }
    }

    function sampleRowsSQL(table) {
      const columns = Array.isArray(table.columns) && table.columns.length > 0 ? table.columns : ["*"];
      const projection = columns.includes("*") ? "*" : columns.slice(0, 8).join(", ");
      const orderColumn = columns.includes("window_start") ? "window_start" : columns.includes("ts") ? "ts" : "";
      const orderBy = orderColumn ? " order by " + orderColumn + " desc" : "";
      return "select " + projection + " from " + table.name + orderBy + " limit 20";
    }

    async function loadMetadata() {
      if (!(await ensureHostedReady())) return;
      statusEl.textContent = "Loading metadata...";
      const response = await fetch(apiPath("/api/v1/metadata"));
      const payload = await response.json();
      if (!response.ok) {
        metricsEl.innerHTML = "<p>Metadata is not available for this source yet.</p>";
        statusEl.textContent = payload.message || "Metadata request failed.";
        return;
      }
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

    async function loadCapabilities() {
      if (!(await ensureHostedReady())) return;
      const response = await fetch(apiPath("/capabilities"));
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        capabilities = { mode: activeSource.mode, storage: "unavailable", auth: !!activeSource.auth, features: { metadata: false, query: false, queryRange: false, sql: false }, schema: [] };
        sqlStatusEl.textContent = payload.message || "Metrics source is not available yet.";
        runSQLEl.disabled = true;
        renderSQLExamples([], false);
        schemaEl.textContent = JSON.stringify(payload, null, 2);
        return;
      }
      capabilities = await response.json();
      const features = capabilities.features || {};
      capabilitiesEl.innerHTML = "";
      for (const item of ["mode", "storage", "auth"]) {
        const span = document.createElement("span");
        span.className = "pill";
        span.textContent = item + ": " + capabilities[item];
        capabilitiesEl.appendChild(span);
      }
      sqlCardEl.classList.toggle("disabled", !features.sql);
      sqlStatusEl.textContent = features.sql
        ? "SQL is available for this backend."
        : "SQL is not available for this backend yet. This panel will activate for local SQLite and future authenticated Durable Object SQL backends.";
      runSQLEl.disabled = !features.sql;
      schemaEl.textContent = JSON.stringify(capabilities.schema || [], null, 2);
      renderSQLExamples(capabilities.schema || [], features.sql);
    }

    async function runSQL() {
      if (!capabilities || !capabilities.features || !capabilities.features.sql) {
        sqlStatusEl.textContent = "SQL is not available for this backend.";
        return;
      }
      sqlStatusEl.textContent = "Running SQL...";
      const response = await fetch(apiPath("/sql"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: sqlQueryEl.value, limit: 1000 }),
      });
      const payload = await response.json();
      sqlOutputEl.textContent = JSON.stringify(payload, null, 2);
      if (response.ok) {
        renderSQLTable(payload);
        const stats = payload.stats || {};
        sqlStatusEl.textContent = "SQL complete. " + (stats.rowCount || 0) + " rows in " + (stats.durationMs || 0) + "ms.";
        setSQLView("table");
      } else {
        sqlTablePaneEl.innerHTML = '<p class="empty">SQL failed. Open Raw JSON for details.</p>';
        sqlStatusEl.textContent = "SQL failed.";
        setSQLView("raw");
      }
    }

    async function runQuery() {
      const query = queryEl.value.trim();
      if (!query) {
        statusEl.textContent = "Enter a query first.";
        return;
      }
      statusEl.textContent = "Running query...";
      const response = await fetch(apiPath("/api/v1/query") + "?query=" + encodeURIComponent(query));
      const payload = await response.json();
      renderJSON(payload);
      statusEl.textContent = response.ok ? "Query complete." : "Query failed.";
    }

    document.querySelector("#refresh").addEventListener("click", loadMetadata);
    document.querySelector("#run").addEventListener("click", runQuery);
    runSQLEl.addEventListener("click", runSQL);
    sqlTableTabEl.addEventListener("click", () => setSQLView("table"));
    sqlRawTabEl.addEventListener("click", () => setSQLView("raw"));
    queryEl.addEventListener("keydown", (event) => {
      if (event.key === "Enter") runQuery();
    });
    renderSources();
    Promise.all([loadCapabilities(), loadMetadata()]).catch((error) => {
      statusEl.textContent = "Metadata request failed.";
      outputEl.textContent = error instanceof Error ? error.message : String(error);
    });
  </script>
</body>
</html>`
}
