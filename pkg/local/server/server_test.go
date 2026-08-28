package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	localmetrics "github.com/distlanglabs/distlang/pkg/local/metrics"
)

func TestMetricsMetadataAndQuery(t *testing.T) {
	store := localmetrics.NewMemoryStore()
	r := &Running{store: store}
	ts := time.Now().UTC().Add(-time.Second).Truncate(time.Second)

	putJSON(t, r, http.MethodPut, "/distlang/metrics/v1/metricsets/local-app", nil)
	putJSON(t, r, http.MethodPut, "/distlang/metrics/v1/metricsets/local-app/metadata", map[string]any{
		"metrics": map[string]any{"trafficReqCount": map[string]any{"kind": "counter", "description": "Traffic", "unit": "count"}},
	})
	putJSON(t, r, http.MethodPost, "/distlang/metrics/v1/metricsets/local-app/rows", map[string]any{
		"rows": []map[string]any{{"ts": ts.Format(time.RFC3339), "data": map[string]any{"metric": "trafficReqCount", "kind": "counter", "windowStart": ts.Format(time.RFC3339), "count": 3, "sum": 3}}},
	})

	metadataReq := httptest.NewRequest(http.MethodGet, "/distlang/metrics/v1/api/v1/metadata", nil)
	metadataRes := httptest.NewRecorder()
	r.handle(metadataRes, metadataReq)
	if metadataRes.Code != http.StatusOK {
		t.Fatalf("metadata status: %d %s", metadataRes.Code, metadataRes.Body.String())
	}

	queryReq := httptest.NewRequest(http.MethodGet, `/distlang/metrics/v1/api/v1/query?query=increase(trafficReqCount{metricSet="local-app"}[30s])`, nil)
	queryRes := httptest.NewRecorder()
	r.handle(queryRes, queryReq)
	if queryRes.Code != http.StatusOK {
		t.Fatalf("query status: %d %s", queryRes.Code, queryRes.Body.String())
	}
	var payload struct {
		Data struct {
			Result []struct {
				Value []any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(queryRes.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode query: %v", err)
	}
	if len(payload.Data.Result) != 1 || payload.Data.Result[0].Value[1] != "3" {
		t.Fatalf("unexpected query payload: %s", queryRes.Body.String())
	}
}

func TestMetricsExplorerHTML(t *testing.T) {
	r := &Running{store: localmetrics.NewMemoryStore()}
	req := httptest.NewRequest(http.MethodGet, "/distlang/metrics", nil)
	res := httptest.NewRecorder()
	r.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"Local Metrics Explorer", "/distlang/metrics/v1/api/v1/metadata", "/distlang/metrics/v1/capabilities", "sql-table-pane", "Table", "Raw JSON"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics explorer html missing %q: %s", expected, body)
		}
	}
}

func TestCapabilitiesEndpoint(t *testing.T) {
	r := &Running{store: localmetrics.NewMemoryStore()}
	req := httptest.NewRequest(http.MethodGet, "/distlang/metrics/v1/capabilities", nil)
	res := httptest.NewRecorder()
	r.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}
	var payload localmetrics.Capabilities
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if payload.Mode != "local" || payload.Storage != "memory" || payload.Auth || payload.Features["sql"] {
		t.Fatalf("unexpected capabilities: %#v", payload)
	}
}

func TestSQLEndpointUnavailableForMemoryBackend(t *testing.T) {
	r := &Running{store: localmetrics.NewMemoryStore()}
	req := httptest.NewRequest(http.MethodPost, "/distlang/metrics/v1/sql", strings.NewReader(`{"query":"select 1"}`))
	res := httptest.NewRecorder()
	r.handle(res, req)
	if res.Code != http.StatusNotImplemented {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}
}

func TestSQLEndpointWithSQLiteBackend(t *testing.T) {
	store, err := localmetrics.OpenSQLiteStore(t.TempDir() + "/metrics.db")
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}
	defer store.Close()
	r := &Running{store: store, backend: localmetrics.NewSQLiteQueryBackend(store)}
	putJSON(t, r, http.MethodPut, "/distlang/metrics/v1/metricsets/local-app/metadata", map[string]any{
		"metrics": map[string]any{"requests": map[string]any{"kind": "counter", "description": "Requests", "unit": "count"}},
	})
	now := time.Now().UTC().Format(time.RFC3339Nano)
	putJSON(t, r, http.MethodPost, "/distlang/metrics/v1/metricsets/local-app/rows", map[string]any{
		"rows": []map[string]any{{"ts": now, "data": map[string]any{"metric": "requests", "kind": "counter", "windowStart": now, "count": 2, "sum": 2}}},
	})
	req := httptest.NewRequest(http.MethodPost, "/distlang/metrics/v1/sql", strings.NewReader(`{"query":"select metric_set, metric, sum from metric_rows where metric_set = 'local-app'","limit":10}`))
	res := httptest.NewRecorder()
	r.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}
	var payload localmetrics.SQLQueryResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode sql: %v", err)
	}
	if len(payload.Rows) != 1 || payload.Rows[0][0] != "local-app" || payload.Rows[0][1] != "requests" {
		t.Fatalf("unexpected SQL payload: %#v", payload)
	}

	req = httptest.NewRequest(http.MethodPost, "/distlang/metrics/v1/sql", strings.NewReader(`{"query":"show tables"}`))
	res = httptest.NewRecorder()
	r.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("show tables status: %d %s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode show tables: %v", err)
	}
	if len(payload.Rows) != 4 || payload.Rows[0][0] != "metric_definitions" {
		t.Fatalf("unexpected show tables payload: %#v", payload)
	}
}

func putJSON(t *testing.T, r *Running, method string, path string, value any) {
	t.Helper()
	var body bytes.Buffer
	if value != nil {
		if err := json.NewEncoder(&body).Encode(value); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	rec := httptest.NewRecorder()
	r.handle(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("%s %s status: %d %s", method, path, rec.Code, rec.Body.String())
	}
}
