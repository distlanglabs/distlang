package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	session := Session{
		AuthBaseURL:  "https://auth.example.com",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Unix(1700000000, 0).UTC(),
		User: User{
			Email: "test@example.com",
			Name:  "Test User",
		},
	}

	if err := SaveSession(session); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}
	if loaded.AuthBaseURL != session.AuthBaseURL || loaded.AccessToken != session.AccessToken || loaded.RefreshToken != session.RefreshToken {
		t.Fatalf("loaded session mismatch: %+v", loaded)
	}
	if !loaded.ExpiresAt.Equal(session.ExpiresAt) {
		t.Fatalf("expected expires_at %s, got %s", session.ExpiresAt, loaded.ExpiresAt)
	}

	path, err := SessionPath()
	if err != nil {
		t.Fatalf("SessionPath error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
	if filepath.Base(path) != "auth.json" {
		t.Fatalf("unexpected session path: %s", path)
	}
}

func TestLoadSessionMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := LoadSession()
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn, got %v", err)
	}
}

func TestClearSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveSession(Session{AccessToken: "token", RefreshToken: "refresh", ExpiresAt: time.Now().UTC()}); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}
	if err := ClearSession(); err != nil {
		t.Fatalf("ClearSession error: %v", err)
	}
	_, err := LoadSession()
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn after clear, got %v", err)
	}
}
