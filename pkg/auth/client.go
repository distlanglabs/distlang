package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type WhoAmIResponse struct {
	User  User `json:"user"`
	Token struct {
		Scope     string `json:"scope"`
		ExpiresAt string `json:"expires_at"`
	} `json:"token"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = ResolveBaseURL()
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) ExchangeCLIAuthCode(code, state, codeVerifier, redirectURI string) (Session, error) {
	body := map[string]string{
		"code":          code,
		"state":         state,
		"code_verifier": codeVerifier,
		"redirect_uri":  redirectURI,
	}

	var response struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		User         User   `json:"user"`
	}
	if err := c.postJSON("/auth/cli/exchange", body, &response, ""); err != nil {
		return Session{}, err
	}

	return Session{
		AuthBaseURL:  c.baseURL,
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(response.ExpiresIn) * time.Second).UTC(),
		User:         response.User,
	}, nil
}

func (c *Client) Refresh(session Session) (Session, error) {
	body := map[string]string{"refresh_token": session.RefreshToken}
	var response struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.postJSON("/auth/refresh", body, &response, ""); err != nil {
		return Session{}, err
	}

	session.AccessToken = response.AccessToken
	session.RefreshToken = response.RefreshToken
	session.ExpiresAt = time.Now().Add(time.Duration(response.ExpiresIn) * time.Second).UTC()
	session.AuthBaseURL = c.baseURL
	return session, nil
}

func (c *Client) WhoAmI(accessToken string) (WhoAmIResponse, error) {
	var response WhoAmIResponse
	if err := c.getJSON("/auth/whoami", &response, accessToken); err != nil {
		return WhoAmIResponse{}, err
	}
	return response, nil
}

func (c *Client) Logout(refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	return c.postJSON("/auth/logout", map[string]string{"refresh_token": refreshToken}, &struct{}{}, "")
}

func (c *Client) EnsureSession() (Session, error) {
	session, err := LoadSession()
	if err != nil {
		return Session{}, err
	}

	if session.AuthBaseURL == "" {
		session.AuthBaseURL = c.baseURL
	}
	if session.AuthBaseURL != c.baseURL {
		return Session{}, fmt.Errorf("saved session uses %s but current auth base is %s; log in again", session.AuthBaseURL, c.baseURL)
	}

	if !session.NeedsRefresh(time.Now()) {
		return session, nil
	}

	refreshed, err := c.Refresh(session)
	if err != nil {
		_ = ClearSession()
		return Session{}, fmt.Errorf("refresh session: %w", err)
	}
	if err := SaveSession(refreshed); err != nil {
		return Session{}, err
	}
	return refreshed, nil
}

func (c *Client) LogoutAndClear() error {
	session, err := LoadSession()
	if err != nil {
		if err == ErrNotLoggedIn {
			return nil
		}
		return err
	}

	if session.AuthBaseURL != "" && session.AuthBaseURL != c.baseURL {
		c = NewClient(session.AuthBaseURL)
	}
	logoutErr := c.Logout(session.RefreshToken)
	clearErr := ClearSession()
	if logoutErr != nil && clearErr != nil {
		return fmt.Errorf("logout: %v; clear session: %w", logoutErr, clearErr)
	}
	if logoutErr != nil {
		return fmt.Errorf("logout: %w", logoutErr)
	}
	if clearErr != nil {
		return clearErr
	}
	return nil
}

func (c *Client) getJSON(path string, out any, accessToken string) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	return c.do(req, out)
}

func (c *Client) postJSON(path string, body any, out any, accessToken string) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		var payload struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &payload) == nil {
			if payload.Error != "" && payload.Message != "" {
				message = payload.Error + ": " + payload.Message
			} else if payload.Error != "" {
				message = payload.Error
			} else if payload.Message != "" {
				message = payload.Message
			}
		}
		if message == "" {
			message = res.Status
		}
		return fmt.Errorf("auth request failed (%s): %s", res.Status, message)
	}

	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
