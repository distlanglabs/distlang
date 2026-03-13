package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrNotLoggedIn = errors.New("not logged in; run `distlang helpers login`")

type Session struct {
	AuthBaseURL  string    `json:"auth_base_url"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

func SessionPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "distlang", "auth.json"), nil
}

func LoadSession() (Session, error) {
	path, err := SessionPath()
	if err != nil {
		return Session{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, ErrNotLoggedIn
		}
		return Session{}, fmt.Errorf("read session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("decode session: %w", err)
	}
	if session.AuthBaseURL == "" {
		session.AuthBaseURL = ResolveBaseURL()
	}
	return session, nil
}

func SaveSession(session Session) error {
	path, err := SessionPath()
	if err != nil {
		return err
	}
	if session.AuthBaseURL == "" {
		session.AuthBaseURL = ResolveBaseURL()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

func ClearSession() error {
	path, err := SessionPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}

func (s Session) NeedsRefresh(now time.Time) bool {
	if s.AccessToken == "" || s.RefreshToken == "" {
		return true
	}
	if s.ExpiresAt.IsZero() {
		return true
	}
	return !s.ExpiresAt.After(now.Add(30 * time.Second))
}
