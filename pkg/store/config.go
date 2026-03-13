package store

import (
	"os"
	"strings"
)

const defaultBaseURL = "https://api.distlang.com"

func ResolveBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("DISTLANG_STORE_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultBaseURL
}
