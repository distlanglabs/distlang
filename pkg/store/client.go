package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	Analytics  *AnalyticsClient
}

type AnalyticsClient struct {
	client  *Client
	Buckets *AnalyticsBucketsClient
}

type AnalyticsBucketsClient struct {
	client *AnalyticsClient
}

type APIError struct {
	StatusCode int
	Status     string
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if e.Code != "" && message != "" {
		return fmt.Sprintf("store request failed (%s): %s: %s", e.Status, e.Code, message)
	}
	if e.Code != "" {
		return fmt.Sprintf("store request failed (%s): %s", e.Status, e.Code)
	}
	if message != "" {
		return fmt.Sprintf("store request failed (%s): %s", e.Status, message)
	}
	return fmt.Sprintf("store request failed (%s)", e.Status)
}

type BucketRecord struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type ValueMetadata struct {
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	UpdatedAt   string `json:"updatedAt"`
}

type KeyRecord struct {
	Name       string         `json:"name"`
	Expiration any            `json:"expiration"`
	Metadata   *ValueMetadata `json:"metadata"`
}

type ServiceIndex struct {
	OK      bool   `json:"ok"`
	Service string `json:"service"`
	Version string `json:"version"`
	User    struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
	Routes struct {
		Buckets string `json:"buckets"`
		Values  string `json:"values"`
		Keys    string `json:"keys"`
	} `json:"routes"`
}

type ListBucketsResponse struct {
	OK      bool           `json:"ok"`
	Buckets []BucketRecord `json:"buckets"`
}

type CreateBucketResponse struct {
	OK      bool   `json:"ok"`
	Bucket  string `json:"bucket"`
	Created bool   `json:"created"`
}

type DeleteResponse struct {
	OK      bool   `json:"ok"`
	Bucket  string `json:"bucket"`
	Key     string `json:"key"`
	Deleted bool   `json:"deleted"`
}

type PutValueResponse struct {
	OK       bool          `json:"ok"`
	Bucket   string        `json:"bucket"`
	Key      string        `json:"key"`
	Metadata ValueMetadata `json:"metadata"`
}

type ListKeysResponse struct {
	OK           bool        `json:"ok"`
	Bucket       string      `json:"bucket"`
	Keys         []KeyRecord `json:"keys"`
	ListComplete bool        `json:"list_complete"`
	Cursor       string      `json:"cursor"`
}

type GetValueResponse struct {
	Body        []byte
	ContentType string
	Size        string
	UpdatedAt   string
}

type HeadValueResponse struct {
	ContentType string
	ContentSize string
	UpdatedAt   string
}

type ListKeysOptions struct {
	Prefix string
	Limit  int
	Cursor string
}

type AnalyticsServiceIndex struct {
	OK      bool   `json:"ok"`
	Service string `json:"service"`
	Version string `json:"version"`
	User    struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
	Limits struct {
		MaxRowsPerWrite int `json:"maxRowsPerWrite"`
		MaxQueryLimit   int `json:"maxQueryLimit"`
	} `json:"limits"`
	Routes struct {
		Buckets string `json:"buckets"`
		Rows    string `json:"rows"`
		Query   string `json:"query"`
	} `json:"routes"`
}

type AnalyticsCreateBucketResponse struct {
	OK      bool   `json:"ok"`
	Bucket  string `json:"bucket"`
	Created bool   `json:"created"`
}

type AnalyticsPutResponse struct {
	OK      bool   `json:"ok"`
	Bucket  string `json:"bucket"`
	Written int    `json:"written"`
}

type AnalyticsRow struct {
	TS   string `json:"ts"`
	Data any    `json:"data"`
}

type AnalyticsQueryOptions struct {
	From   string
	To     string
	Limit  int
	Cursor string
}

type AnalyticsQueryResponse struct {
	OK         bool           `json:"ok"`
	Bucket     string         `json:"bucket"`
	Rows       []AnalyticsRow `json:"rows"`
	NextCursor string         `json:"next_cursor"`
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = ResolveBaseURL()
	}
	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	analytics := &AnalyticsClient{client: client}
	analytics.Buckets = &AnalyticsBucketsClient{client: analytics}
	client.Analytics = analytics
	return client
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) ObjectDBStatus(accessToken string) (ServiceIndex, error) {
	var response ServiceIndex
	if err := c.getJSON("/objectdb/v1", &response, accessToken); err != nil {
		return ServiceIndex{}, err
	}
	return response, nil
}

func (c *Client) ListBuckets(accessToken string) (ListBucketsResponse, error) {
	var response ListBucketsResponse
	if err := c.getJSON("/objectdb/v1/buckets", &response, accessToken); err != nil {
		return ListBucketsResponse{}, err
	}
	return response, nil
}

func (c *Client) CreateBucket(accessToken, bucket string) (CreateBucketResponse, error) {
	var response CreateBucketResponse
	if err := c.doJSON(http.MethodPut, objectDBBucketPath(bucket), nil, &response, accessToken, ""); err != nil {
		return CreateBucketResponse{}, err
	}
	return response, nil
}

func (c *Client) BucketExists(accessToken, bucket string) (bool, error) {
	response, err := c.ListBuckets(accessToken)
	if err != nil {
		return false, err
	}
	for _, item := range response.Buckets {
		if item.Name == bucket {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) DeleteBucket(accessToken, bucket string) (DeleteResponse, error) {
	var response DeleteResponse
	if err := c.doJSON(http.MethodDelete, objectDBBucketPath(bucket), nil, &response, accessToken, ""); err != nil {
		return DeleteResponse{}, err
	}
	return response, nil
}

func (c *Client) ListKeys(accessToken, bucket string, opts ListKeysOptions) (ListKeysResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(opts.Prefix) != "" {
		query.Set("prefix", opts.Prefix)
	}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}
	if strings.TrimSpace(opts.Cursor) != "" {
		query.Set("cursor", opts.Cursor)
	}

	path := objectDBKeysPath(bucket)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var response ListKeysResponse
	if err := c.getJSON(path, &response, accessToken); err != nil {
		return ListKeysResponse{}, err
	}
	return response, nil
}

func (c *Client) PutValue(accessToken, bucket, key string, body []byte, contentType string) (PutValueResponse, error) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	var response PutValueResponse
	if err := c.doJSON(http.MethodPut, objectDBValuePath(bucket, key), body, &response, accessToken, contentType); err != nil {
		return PutValueResponse{}, err
	}
	return response, nil
}

func (c *Client) GetValue(accessToken, bucket, key, responseType string) (GetValueResponse, error) {
	requestPath := objectDBValuePath(bucket, key)
	if strings.TrimSpace(responseType) != "" {
		requestPath += "?type=" + url.QueryEscape(responseType)
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+requestPath, nil)
	if err != nil {
		return GetValueResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return GetValueResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return GetValueResponse{}, decodeErrorResponse(res)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 26<<20))
	if err != nil {
		return GetValueResponse{}, err
	}
	return GetValueResponse{
		Body:        body,
		ContentType: res.Header.Get("Content-Type"),
		Size:        res.Header.Get("X-Distlang-Value-Size"),
		UpdatedAt:   res.Header.Get("X-Distlang-Updated-At"),
	}, nil
}

func (c *Client) HeadValue(accessToken, bucket, key string) (HeadValueResponse, error) {
	result, err := c.ListKeys(accessToken, bucket, ListKeysOptions{Prefix: key, Limit: 1000})
	if err != nil {
		return HeadValueResponse{}, err
	}
	for _, item := range result.Keys {
		if item.Name != key || item.Metadata == nil {
			continue
		}
		return HeadValueResponse{
			ContentType: item.Metadata.ContentType,
			ContentSize: strconv.FormatInt(item.Metadata.Size, 10),
			UpdatedAt:   item.Metadata.UpdatedAt,
		}, nil
	}
	return HeadValueResponse{}, &APIError{
		StatusCode: http.StatusNotFound,
		Status:     http.StatusText(http.StatusNotFound),
		Code:       "key_not_found",
		Message:    "No value exists for that key.",
	}
}

func (c *Client) DeleteValue(accessToken, bucket, key string) (DeleteResponse, error) {
	var response DeleteResponse
	if err := c.doJSON(http.MethodDelete, objectDBValuePath(bucket, key), nil, &response, accessToken, ""); err != nil {
		return DeleteResponse{}, err
	}
	return response, nil
}

func (a *AnalyticsClient) Status(accessToken string) (AnalyticsServiceIndex, error) {
	var response AnalyticsServiceIndex
	if err := a.client.getJSON("/analyticsdb/v1", &response, accessToken); err != nil {
		return AnalyticsServiceIndex{}, err
	}
	return response, nil
}

func (b *AnalyticsBucketsClient) Create(accessToken, bucket string) (AnalyticsCreateBucketResponse, error) {
	var response AnalyticsCreateBucketResponse
	if err := b.client.client.doJSON(http.MethodPut, analyticsBucketPath(bucket), nil, &response, accessToken, ""); err != nil {
		return AnalyticsCreateBucketResponse{}, err
	}
	return response, nil
}

func (a *AnalyticsClient) Put(accessToken, bucket string, data any) (AnalyticsPutResponse, error) {
	return a.PutAt(accessToken, bucket, time.Now().UTC(), data)
}

func (a *AnalyticsClient) PutAt(accessToken, bucket string, ts time.Time, data any) (AnalyticsPutResponse, error) {
	payload, err := json.Marshal(map[string]any{
		"rows": []map[string]any{{
			"ts":   ts.UTC().Format(time.RFC3339Nano),
			"data": data,
		}},
	})
	if err != nil {
		return AnalyticsPutResponse{}, fmt.Errorf("encode analytics row: %w", err)
	}

	var response AnalyticsPutResponse
	if err := a.client.doJSON(http.MethodPost, analyticsRowsPath(bucket), payload, &response, accessToken, "application/json"); err != nil {
		return AnalyticsPutResponse{}, err
	}
	return response, nil
}

func (a *AnalyticsClient) Query(accessToken, bucket string, opts AnalyticsQueryOptions) (AnalyticsQueryResponse, error) {
	query := url.Values{}
	query.Set("from", strings.TrimSpace(opts.From))
	query.Set("to", strings.TrimSpace(opts.To))
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}
	if strings.TrimSpace(opts.Cursor) != "" {
		query.Set("cursor", opts.Cursor)
	}

	requestPath := analyticsRowsPath(bucket) + "?" + query.Encode()
	var response AnalyticsQueryResponse
	if err := a.client.getJSON(requestPath, &response, accessToken); err != nil {
		return AnalyticsQueryResponse{}, err
	}
	return response, nil
}

func (a *AnalyticsClient) DefaultBucket(appID, env string) string {
	appPart := normalizeBucketPart(appID, "app")
	envPart := normalizeBucketPart(env, "env")
	return trimBucketLength("app_" + appPart + "__" + envPart)
}

func objectDBBucketPath(bucket string) string {
	return path.Join("/objectdb/v1/buckets", url.PathEscape(strings.TrimSpace(bucket)))
}

func objectDBKeysPath(bucket string) string {
	return path.Join("/objectdb/v1/buckets", url.PathEscape(strings.TrimSpace(bucket)), "keys")
}

func objectDBValuePath(bucket, key string) string {
	return path.Join("/objectdb/v1/buckets", url.PathEscape(strings.TrimSpace(bucket)), "values") + "/" + url.PathEscape(key)
}

func analyticsBucketPath(bucket string) string {
	return path.Join("/analyticsdb/v1/buckets", url.PathEscape(strings.TrimSpace(bucket)))
}

func analyticsRowsPath(bucket string) string {
	return path.Join("/analyticsdb/v1/buckets", url.PathEscape(strings.TrimSpace(bucket)), "rows")
}

func (c *Client) getJSON(requestPath string, out any, accessToken string) error {
	return c.doJSON(http.MethodGet, requestPath, nil, out, accessToken, "")
}

func (c *Client) doJSON(method, requestPath string, body []byte, out any, accessToken, contentType string) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.baseURL+requestPath, reader)
	if err != nil {
		return err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	if body != nil {
		if strings.TrimSpace(contentType) == "" {
			contentType = "application/octet-stream"
		}
		req.Header.Set("Content-Type", contentType)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return decodeErrorResponse(res)
	}
	if out == nil {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func decodeErrorResponse(res *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	message := strings.TrimSpace(string(body))
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &payload) == nil {
		if payload.Message != "" {
			message = payload.Message
		}
		return &APIError{
			StatusCode: res.StatusCode,
			Status:     res.Status,
			Code:       payload.Error,
			Message:    message,
		}
	}
	if message == "" {
		message = res.Status
	}
	return &APIError{StatusCode: res.StatusCode, Status: res.Status, Message: message}
}

func normalizeBucketPart(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case r == '_' || r == '-':
			if b.Len() == 0 || lastUnderscore {
				continue
			}
			b.WriteByte('_')
			lastUnderscore = true
		default:
			if b.Len() == 0 || lastUnderscore {
				continue
			}
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(b.String(), "_")
	if result == "" {
		return fallback
	}
	return result
}

func trimBucketLength(value string) string {
	if len(value) <= 64 {
		return value
	}
	trimmed := value[:64]
	trimmed = strings.Trim(trimmed, "_")
	if trimmed == "" {
		return "analytics_bucket"
	}
	return trimmed
}
