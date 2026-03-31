package mockserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Host string
	Port int
}

type Running struct {
	server *http.Server
	ln     net.Listener
	store  *objectStore
}

func Start(cfg Config) (*Running, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port < 0 {
		port = 9191
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}

	r := &Running{ln: ln, store: newObjectStore()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", r.handle)
	r.server = &http.Server{Handler: mux}

	go func() {
		_ = r.server.Serve(ln)
	}()

	return r, nil
}

func (r *Running) URL() string {
	return "http://" + r.ln.Addr().String()
}

func (r *Running) Close(ctx context.Context) error {
	return r.server.Shutdown(ctx)
}

type objectStore struct {
	mu      sync.RWMutex
	buckets map[string]map[string]storedValue
}

type storedValue struct {
	body        []byte
	contentType string
	updatedAt   time.Time
}

func newObjectStore() *objectStore {
	return &objectStore{buckets: map[string]map[string]storedValue{}}
}

func (r *Running) handle(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	if req.Method == http.MethodGet && path == "/objectdb/v1" {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"service": "objectdb",
			"version": "mock",
			"user": map[string]any{
				"id":    "local",
				"email": "local@distlang",
				"name":  "Distlang Local Mock",
			},
			"routes": map[string]string{
				"buckets": "/objectdb/v1/buckets",
				"values":  "/objectdb/v1/buckets/{bucket}/values/{key}",
				"keys":    "/objectdb/v1/buckets/{bucket}/keys",
			},
		})
		return
	}

	if req.Method == http.MethodGet && path == "/objectdb/v1/buckets" {
		r.handleListBuckets(w)
		return
	}

	if strings.HasPrefix(path, "/objectdb/v1/buckets/") {
		r.handleBucketRoutes(w, req)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "route not found"})
}

func (r *Running) handleListBuckets(w http.ResponseWriter) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	buckets := make([]map[string]string, 0, len(r.store.buckets))
	for name := range r.store.buckets {
		buckets = append(buckets, map[string]string{"name": name, "createdAt": ""})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "buckets": buckets})
}

func (r *Running) handleBucketRoutes(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(strings.TrimPrefix(req.URL.Path, "/objectdb/v1/buckets/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_bucket", "message": "bucket required"})
		return
	}
	bucket, _ := url.PathUnescape(parts[0])

	if len(parts) == 1 {
		switch req.Method {
		case http.MethodPut:
			r.store.mu.Lock()
			_, existed := r.store.buckets[bucket]
			if !existed {
				r.store.buckets[bucket] = map[string]storedValue{}
			}
			r.store.mu.Unlock()
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": bucket, "created": !existed})
			return
		case http.MethodDelete:
			r.store.mu.Lock()
			_, existed := r.store.buckets[bucket]
			delete(r.store.buckets, bucket)
			r.store.mu.Unlock()
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": bucket, "key": "", "deleted": existed})
			return
		default:
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "route not found"})
			return
		}
	}

	if len(parts) >= 2 && parts[1] == "keys" && req.Method == http.MethodGet {
		r.handleListKeys(w, req, bucket)
		return
	}

	if len(parts) >= 3 && parts[1] == "values" {
		key, _ := url.PathUnescape(strings.Join(parts[2:], "/"))
		r.handleValue(w, req, bucket, key)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "route not found"})
}

func (r *Running) handleListKeys(w http.ResponseWriter, req *http.Request, bucket string) {
	prefix := req.URL.Query().Get("prefix")
	limit := 1000
	if raw := strings.TrimSpace(req.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	r.store.mu.RLock()
	items, ok := r.store.buckets[bucket]
	r.store.mu.RUnlock()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": bucket, "keys": []any{}, "list_complete": true, "cursor": ""})
		return
	}

	keys := []map[string]any{}
	for key, value := range items {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		keys = append(keys, map[string]any{
			"name":       key,
			"expiration": nil,
			"metadata": map[string]any{
				"contentType": value.contentType,
				"size":        len(value.body),
				"updatedAt":   value.updatedAt.UTC().Format(time.RFC3339),
			},
		})
		if len(keys) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": bucket, "keys": keys, "list_complete": true, "cursor": ""})
}

func (r *Running) handleValue(w http.ResponseWriter, req *http.Request, bucket, key string) {
	switch req.Method {
	case http.MethodPut:
		r.store.mu.Lock()
		defer r.store.mu.Unlock()
		items, ok := r.store.buckets[bucket]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "bucket_not_found", "message": "bucket does not exist"})
			return
		}
		body, err := io.ReadAll(io.LimitReader(req.Body, 26<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_body", "message": err.Error()})
			return
		}
		contentType := strings.TrimSpace(req.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		value := storedValue{body: body, contentType: contentType, updatedAt: time.Now().UTC()}
		items[key] = value
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"bucket": bucket,
			"key":    key,
			"metadata": map[string]any{
				"contentType": value.contentType,
				"size":        len(value.body),
				"updatedAt":   value.updatedAt.Format(time.RFC3339),
			},
		})
		return
	case http.MethodGet:
		r.store.mu.RLock()
		items, ok := r.store.buckets[bucket]
		value, exists := items[key]
		r.store.mu.RUnlock()
		if !ok || !exists {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "key_not_found", "message": "No value exists for that key."})
			return
		}
		responseType := strings.TrimSpace(req.URL.Query().Get("type"))
		if responseType == "" {
			responseType = "json"
		}
		w.Header().Set("X-Distlang-Value-Size", strconv.Itoa(len(value.body)))
		w.Header().Set("X-Distlang-Updated-At", value.updatedAt.Format(time.RFC3339))
		switch responseType {
		case "json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(value.body)
		case "bytes":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(value.body)
		default:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write(value.body)
		}
		return
	case http.MethodDelete:
		r.store.mu.Lock()
		items, ok := r.store.buckets[bucket]
		if !ok {
			r.store.mu.Unlock()
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": bucket, "key": key, "deleted": false})
			return
		}
		_, existed := items[key]
		delete(items, key)
		r.store.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "bucket": bucket, "key": key, "deleted": existed})
		return
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "route not found"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (r *Running) String() string {
	return fmt.Sprintf("mockserver(%s)", r.URL())
}
