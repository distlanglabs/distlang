package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientEnsureSessionRefreshesAndSaves(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/refresh" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"expires_in":    900,
		})
	}))
	defer server.Close()

	if err := SaveSession(Session{
		AuthBaseURL:  server.URL,
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}

	client := NewClient(server.URL)
	session, err := client.EnsureSession()
	if err != nil {
		t.Fatalf("EnsureSession error: %v", err)
	}
	if session.AccessToken != "new-access" || session.RefreshToken != "new-refresh" {
		t.Fatalf("unexpected refreshed session: %+v", session)
	}

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}
	if loaded.AccessToken != "new-access" {
		t.Fatalf("expected saved access token to refresh, got %s", loaded.AccessToken)
	}
}

func TestClientWhoAmI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/whoami" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":  map[string]any{"id": "usr_123", "email": "test@example.com", "name": "Test User"},
			"token": map[string]any{"scope": "user", "expires_at": "2026-03-13T00:00:00Z"},
		})
	}))
	defer server.Close()

	identity, err := NewClient(server.URL).WhoAmI("access-token")
	if err != nil {
		t.Fatalf("WhoAmI error: %v", err)
	}
	if identity.User.Email != "test@example.com" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestBuildLoginURL(t *testing.T) {
	url := buildLoginURL("https://auth.example.com/", "state-123", "challenge-456")
	if url != "https://auth.example.com/auth/google/start?code_challenge=challenge-456&redirect_uri=http%3A%2F%2F127.0.0.1%3A8976%2Fcallback&state=state-123" {
		t.Fatalf("unexpected login URL: %s", url)
	}
}
