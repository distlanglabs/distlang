package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type LoginResult struct {
	Session Session
	User    User
}

type loginCallback struct {
	Code  string
	State string
	Err   string
}

func Login() (LoginResult, error) {
	client := NewClient(ResolveBaseURL())
	state, err := randomBase64URL(32)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate state: %w", err)
	}
	codeVerifier, err := randomBase64URL(32)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate code verifier: %w", err)
	}
	codeChallenge := pkceChallenge(codeVerifier)

	callbackCh := make(chan loginCallback, 1)
	errCh := make(chan error, 1)
	server, err := startCallbackServer(callbackCh, errCh)
	if err != nil {
		return LoginResult{}, err
	}
	defer shutdownServer(server)

	loginURL := buildLoginURL(client.baseURL, state, codeChallenge)
	if err := openBrowser(loginURL); err != nil {
		fmt.Printf("Open this URL to continue login:\n%s\n", loginURL)
	} else {
		fmt.Println("Waiting for browser login to complete...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var callback loginCallback
	select {
	case callback = <-callbackCh:
	case err := <-errCh:
		return LoginResult{}, err
	case <-ctx.Done():
		return LoginResult{}, fmt.Errorf("timed out waiting for login callback")
	}

	if callback.Err != "" {
		return LoginResult{}, fmt.Errorf("login failed: %s", callback.Err)
	}
	if callback.State != state {
		return LoginResult{}, fmt.Errorf("login failed: state mismatch")
	}

	session, err := client.ExchangeCLIAuthCode(callback.Code, state, codeVerifier, CallbackURL())
	if err != nil {
		return LoginResult{}, err
	}
	if err := SaveSession(session); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Session: session, User: session.User}, nil
}

func buildLoginURL(baseURL, state, codeChallenge string) string {
	u, _ := url.Parse(strings.TrimRight(baseURL, "/") + "/auth/google/start")
	query := u.Query()
	query.Set("state", state)
	query.Set("redirect_uri", CallbackURL())
	query.Set("code_challenge", codeChallenge)
	u.RawQuery = query.Encode()
	return u.String()
}

func startCallbackServer(callbackCh chan<- loginCallback, errCh chan<- error) (*http.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:8976")
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", CallbackURL(), err)
	}

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		callback := loginCallback{
			Code:  strings.TrimSpace(r.URL.Query().Get("code")),
			State: strings.TrimSpace(r.URL.Query().Get("state")),
			Err:   strings.TrimSpace(r.URL.Query().Get("error")),
		}
		if callback.Code == "" && callback.Err == "" {
			callback.Err = "missing code"
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Login complete. You can return to distlang now.\n"))

		select {
		case callbackCh <- callback:
		default:
		}
		go shutdownServer(server)
	})

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()

	return server, nil
}

func shutdownServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomBase64URL(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
