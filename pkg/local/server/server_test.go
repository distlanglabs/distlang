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
	if !strings.Contains(body, "Local Metrics Explorer") || !strings.Contains(body, "/distlang/metrics/v1/api/v1/metadata") {
		t.Fatalf("metrics explorer html missing expected content: %s", body)
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
