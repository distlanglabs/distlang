package metrics

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type QueryBackend interface {
	Capabilities(ctx context.Context) Capabilities
	Metadata(ctx context.Context, req MetadataRequest) (MetadataResponse, error)
	Query(ctx context.Context, req QueryRequest) (QueryResponse, error)
	QueryRange(ctx context.Context, req QueryRangeRequest) (QueryRangeResponse, error)
	SQL(ctx context.Context, req SQLQueryRequest) (SQLQueryResponse, error)
}

type Capabilities struct {
	Mode     string            `json:"mode"`
	Storage  string            `json:"storage"`
	Auth     bool              `json:"auth"`
	Features map[string]bool   `json:"features"`
	Limits   map[string]any    `json:"limits,omitempty"`
	Schema   []SQLSchemaObject `json:"schema,omitempty"`
}

type SQLSchemaObject struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Columns     []string `json:"columns"`
}

type MetadataRequest struct {
	Metric string `json:"metric,omitempty"`
}

type MetadataResponse struct {
	Status string                     `json:"status"`
	Data   map[string][]MetadataEntry `json:"data"`
}

type QueryRequest struct {
	Query string    `json:"query"`
	Time  time.Time `json:"time,omitempty"`
}

type QueryRangeRequest struct {
	Query string        `json:"query"`
	Start time.Time     `json:"start"`
	End   time.Time     `json:"end"`
	Step  time.Duration `json:"step"`
}

type QueryResponse struct {
	Status string          `json:"status"`
	Data   QueryVectorData `json:"data"`
}

type QueryVectorData struct {
	ResultType string         `json:"resultType"`
	Result     []VectorResult `json:"result"`
}

type VectorResult struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

type QueryRangeResponse struct {
	Status string          `json:"status"`
	Data   QueryMatrixData `json:"data"`
}

type QueryMatrixData struct {
	ResultType string         `json:"resultType"`
	Result     []MatrixResult `json:"result"`
}

type MatrixResult struct {
	Metric map[string]string `json:"metric"`
	Values []any             `json:"values"`
}

type SQLQueryRequest struct {
	Query  string `json:"query"`
	Params []any  `json:"params,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type SQLQueryResponse struct {
	Columns []SQLColumn   `json:"columns"`
	Rows    [][]any       `json:"rows"`
	Stats   SQLQueryStats `json:"stats"`
}

type SQLColumn struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type SQLQueryStats struct {
	RowCount   int   `json:"rowCount"`
	DurationMS int64 `json:"durationMs"`
}

type MemoryQueryBackend struct {
	store Store
}

func NewMemoryQueryBackend(store Store) *MemoryQueryBackend {
	return &MemoryQueryBackend{store: store}
}

func (b *MemoryQueryBackend) Capabilities(ctx context.Context) Capabilities {
	return Capabilities{
		Mode:    "local",
		Storage: "memory",
		Auth:    false,
		Features: map[string]bool{
			"metadata":   true,
			"query":      true,
			"queryRange": true,
			"sql":        false,
			"writeRows":  true,
		},
		Limits: map[string]any{
			"sqlReadOnly": true,
			"sqlMaxRows":  1000,
		},
		Schema: []SQLSchemaObject{
			{Name: "metric_definitions", Description: "Future local SQLite metric definitions table", Columns: []string{"metric_set", "metric", "kind", "description", "unit", "labels_json"}},
			{Name: "metric_rows", Description: "Future local SQLite metric rows table", Columns: []string{"metric_set", "metric", "kind", "window_start", "labels_json", "count", "sum"}},
			{Name: "metric_row_values", Description: "Future local SQLite histogram values table", Columns: []string{"metric_set", "metric", "window_start", "value"}},
		},
	}
}

func (b *MemoryQueryBackend) Metadata(ctx context.Context, req MetadataRequest) (MetadataResponse, error) {
	metadata, err := b.store.Metadata(ctx, req.Metric)
	if err != nil {
		return MetadataResponse{}, err
	}
	return MetadataResponse{Status: "success", Data: metadata}, nil
}

func (b *MemoryQueryBackend) Query(ctx context.Context, req QueryRequest) (QueryResponse, error) {
	parsed, err := parsePromQuery(req.Query)
	if err != nil {
		return QueryResponse{}, err
	}
	evalTime := req.Time
	if evalTime.IsZero() {
		evalTime = time.Now().UTC()
	}
	value, err := b.evaluate(ctx, parsed, evalTime)
	if err != nil {
		return QueryResponse{}, err
	}
	result := []VectorResult{}
	if value != 0 {
		result = append(result, VectorResult{
			Metric: map[string]string{"__name__": parsed.Metric, "metricSet": parsed.MetricSet},
			Value:  []any{float64(evalTime.UnixNano()) / 1e9, strconv.FormatFloat(value, 'f', -1, 64)},
		})
	}
	return QueryResponse{Status: "success", Data: QueryVectorData{ResultType: "vector", Result: result}}, nil
}

func (b *MemoryQueryBackend) QueryRange(ctx context.Context, req QueryRangeRequest) (QueryRangeResponse, error) {
	parsed, err := parsePromQuery(req.Query)
	if err != nil {
		return QueryRangeResponse{}, err
	}
	if req.Start.IsZero() || req.End.IsZero() || req.Step <= 0 {
		return QueryRangeResponse{}, errors.New("start, end, and positive step are required")
	}
	values := []any{}
	for ts := req.Start; !ts.After(req.End); ts = ts.Add(req.Step) {
		value, err := b.evaluate(ctx, parsed, ts)
		if err != nil {
			return QueryRangeResponse{}, err
		}
		values = append(values, []any{float64(ts.UnixNano()) / 1e9, strconv.FormatFloat(value, 'f', -1, 64)})
	}
	result := []MatrixResult{}
	if len(values) > 0 {
		result = append(result, MatrixResult{Metric: map[string]string{"__name__": parsed.Metric, "metricSet": parsed.MetricSet}, Values: values})
	}
	return QueryRangeResponse{Status: "success", Data: QueryMatrixData{ResultType: "matrix", Result: result}}, nil
}

func (b *MemoryQueryBackend) SQL(ctx context.Context, req SQLQueryRequest) (SQLQueryResponse, error) {
	return SQLQueryResponse{}, errors.New("SQL queries require a SQL-backed metrics backend")
}

type parsedPromQuery struct {
	Function  string
	Metric    string
	MetricSet string
	Range     time.Duration
}

var promRangeFunctionRE = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\(([a-zA-Z_:][a-zA-Z0-9_:]*)\{metricSet="([^"]+)"\}\[([^\]]+)\]\)$`)

func parsePromQuery(raw string) (parsedPromQuery, error) {
	match := promRangeFunctionRE.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return parsedPromQuery{}, fmt.Errorf("unsupported metrics query: %s", raw)
	}
	if match[1] != "increase" && match[1] != "sum_over_time" {
		return parsedPromQuery{}, fmt.Errorf("unsupported metrics function: %s", match[1])
	}
	duration, err := ParsePromDuration(match[4])
	if err != nil {
		return parsedPromQuery{}, err
	}
	return parsedPromQuery{Function: match[1], Metric: match[2], MetricSet: match[3], Range: duration}, nil
}

func (b *MemoryQueryBackend) evaluate(ctx context.Context, query parsedPromQuery, evalTime time.Time) (float64, error) {
	rows, err := b.store.LoadRows(ctx, RowQuery{MetricSet: query.MetricSet, Metric: query.Metric, Start: evalTime.Add(-query.Range), End: evalTime})
	if err != nil {
		return 0, err
	}
	var total float64
	for _, row := range rows {
		if row.Row.Kind == "histogram" && len(row.Row.Values) > 0 {
			for _, value := range row.Row.Values {
				total += value
			}
			continue
		}
		total += row.Row.Sum
	}
	return total, nil
}

func ParsePromDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("duration is required")
	}
	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(raw)
}

func ParsePromTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("time is required")
	}
	if numeric, err := strconv.ParseFloat(raw, 64); err == nil {
		seconds := int64(numeric)
		nanos := int64((numeric - float64(seconds)) * 1e9)
		return time.Unix(seconds, nanos).UTC(), nil
	}
	return time.Parse(time.RFC3339Nano, raw)
}
