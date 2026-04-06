package deployclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListDeployments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/deployments/v1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}

		_ = json.NewEncoder(w).Encode(ListDeploymentsResponse{
			OK: true,
			Deployments: []DeploymentRecord{{
				ID:             "dep_123",
				App:            "echo",
				MetricsBuckets: []string{"app-echo-metrics"},
				Provider:       "cloudflare",
				ScriptName:     "dl-test-echo",
				Hostname:       "echo-test.distlang.test",
				URL:            "https://echo-test.distlang.test",
				Status:         "published",
				CreatedAt:      "2026-04-06T00:00:00Z",
				UpdatedAt:      "2026-04-06T00:01:00Z",
			}},
		})
	}))
	defer server.Close()

	deployments, err := New(server.URL).ListDeployments("access-token")
	if err != nil {
		t.Fatalf("ListDeployments error: %v", err)
	}
	if len(deployments) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(deployments))
	}
	if deployments[0].ID != "dep_123" || deployments[0].Hostname != "echo-test.distlang.test" {
		t.Fatalf("unexpected deployment: %+v", deployments[0])
	}
}

func TestClientDeleteDeployment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/deployments/v1/dep_123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	if err := New(server.URL).DeleteDeployment("access-token", "dep_123"); err != nil {
		t.Fatalf("DeleteDeployment error: %v", err)
	}
}

func TestClientDeleteDeploymentPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"deployment_not_found"}`))
	}))
	defer server.Close()

	err := New(server.URL).DeleteDeployment("access-token", "dep_missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "delete deployment failed (404 Not Found): deployment_not_found" {
		t.Fatalf("unexpected error: %s", got)
	}
}
