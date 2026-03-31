package store

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnalyticsBucketsCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyticsdb/v1/buckets/app_chat__prod" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "bucket": "app_chat__prod", "created": true})
	}))
	defer server.Close()

	response, err := NewClient(server.URL).Analytics.Buckets.Create("access-token", "app_chat__prod")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if !response.Created || response.Bucket != "app_chat__prod" {
		t.Fatalf("unexpected create response: %+v", response)
	}
}

func TestAnalyticsPutAddsTimestamp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyticsdb/v1/buckets/app_chat__prod/rows" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}
		var body struct {
			Rows []struct {
				TS   string         `json:"ts"`
				Data map[string]any `json:"data"`
			} `json:"rows"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.Rows) != 1 {
			t.Fatalf("expected one row, got %d", len(body.Rows))
		}
		if _, err := time.Parse(time.RFC3339Nano, body.Rows[0].TS); err != nil {
			t.Fatalf("expected RFC3339 timestamp, got %q", body.Rows[0].TS)
		}
		if body.Rows[0].Data["event"] != "http_request" {
			t.Fatalf("unexpected payload: %#v", body.Rows[0].Data)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "bucket": "app_chat__prod", "written": 1})
	}))
	defer server.Close()

	response, err := NewClient(server.URL).Analytics.Put("access-token", "app_chat__prod", map[string]any{"event": "http_request"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	if response.Written != 1 {
		t.Fatalf("unexpected put response: %+v", response)
	}
}

func TestAnalyticsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyticsdb/v1/buckets/app_chat__prod/rows" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("from") != "2026-03-31T11:00:00Z" || query.Get("to") != "2026-03-31T12:00:00Z" {
			t.Fatalf("unexpected time range: %s", r.URL.RawQuery)
		}
		if query.Get("limit") != "10" || query.Get("cursor") != "2" {
			t.Fatalf("unexpected paging query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"bucket":      "app_chat__prod",
			"rows":        []map[string]any{{"ts": "2026-03-31T11:30:00Z", "data": map[string]any{"event": "http_request"}}},
			"next_cursor": "",
		})
	}))
	defer server.Close()

	response, err := NewClient(server.URL).Analytics.Query("access-token", "app_chat__prod", AnalyticsQueryOptions{
		From:   "2026-03-31T11:00:00Z",
		To:     "2026-03-31T12:00:00Z",
		Limit:  10,
		Cursor: "2",
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(response.Rows) != 1 || response.Rows[0].TS != "2026-03-31T11:30:00Z" {
		t.Fatalf("unexpected query response: %+v", response)
	}
}

func TestAnalyticsDefaultBucket(t *testing.T) {
	client := NewClient("https://api.example.com")
	bucket := client.Analytics.DefaultBucket("Chat App", "Prod-US")
	if bucket != "app_chat_app__prod_us" {
		t.Fatalf("unexpected bucket: %s", bucket)
	}
}
