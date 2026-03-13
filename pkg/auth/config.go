package auth

import (
	"os"
	"strings"
)

const (
	defaultBaseURL = "https://auth.distlang.com"
	callbackURL    = "http://127.0.0.1:8976/callback"
)

func ResolveBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("DISTLANG_AUTH_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultBaseURL
}

func CallbackURL() string {
	return callbackURL
}
