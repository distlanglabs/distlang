package mockserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestObjectDBMockCRUD(t *testing.T) {
	r, err := Start(Config{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = r.Close(context.Background()) }()

	client := &http.Client{Timeout: 5 * time.Second}

	req, _ := http.NewRequest(http.MethodPut, r.URL()+"/objectdb/v1/buckets/demo", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create bucket status: %d", res.StatusCode)
	}

	body := []byte(`{"times":3}`)
	req, _ = http.NewRequest(http.MethodPut, r.URL()+"/objectdb/v1/buckets/demo/values/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("put value: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put value status: %d", res.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, r.URL()+"/objectdb/v1/buckets/demo/values/config?type=json", nil)
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("get value: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get value status: %d", res.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["times"] != float64(3) {
		t.Fatalf("expected times=3, got %v", payload["times"])
	}
}
