package metrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const defaultSQLLimit = 1000

type SQLiteStore struct {
	db *sql.DB
}

func DefaultSQLitePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".distlang", "local", "metrics.db"), nil
}

func OpenSQLiteStore(path string) (*SQLiteStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		resolved, err := DefaultSQLitePath()
		if err != nil {
			return nil, err
		}
		path = resolved
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &SQLiteStore{db: db}
	if err := store.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) init(ctx context.Context) error {
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS metric_sets (
			metric_set TEXT PRIMARY KEY,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_definitions (
			metric_set TEXT NOT NULL,
			metric TEXT NOT NULL,
			kind TEXT NOT NULL,
			description TEXT NOT NULL,
			unit TEXT NOT NULL,
			labels_json TEXT NOT NULL,
			PRIMARY KEY (metric_set, metric)
		)`,
		`CREATE TABLE IF NOT EXISTS metric_rows (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metric_set TEXT NOT NULL,
			metric TEXT NOT NULL,
			kind TEXT NOT NULL,
			window_start TEXT NOT NULL,
			labels_json TEXT NOT NULL,
			count INTEGER NOT NULL,
			sum REAL NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_row_values (
			row_id INTEGER NOT NULL,
			value REAL NOT NULL,
			FOREIGN KEY (row_id) REFERENCES metric_rows(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS metric_rows_lookup ON metric_rows(metric_set, metric, window_start)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) EnsureMetricSet(ctx context.Context, metricSet string, definitions map[string]Definition) error {
	metricSet = strings.TrimSpace(metricSet)
	if metricSet == "" {
		return ErrInvalidMetricSet
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO metric_sets(metric_set, created_at) VALUES (?, ?)`, metricSet, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	for name, definition := range definitions {
		labels, err := json.Marshal(definition.Labels)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO metric_definitions(metric_set, metric, kind, description, unit, labels_json) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(metric_set, metric) DO UPDATE SET kind = excluded.kind, description = excluded.description, unit = excluded.unit, labels_json = excluded.labels_json`, metricSet, name, definition.Kind, definition.Description, definition.Unit, string(labels)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) AppendRows(ctx context.Context, metricSet string, rows []Row) error {
	metricSet = strings.TrimSpace(metricSet)
	if metricSet == "" {
		return ErrInvalidMetricSet
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO metric_sets(metric_set, created_at) VALUES (?, ?)`, metricSet, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	for _, row := range rows {
		labels, err := json.Marshal(row.Labels)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO metric_rows(metric_set, metric, kind, window_start, labels_json, count, sum) VALUES (?, ?, ?, ?, ?, ?, ?)`, metricSet, row.Metric, row.Kind, row.WindowStart.UTC().Format(time.RFC3339Nano), string(labels), row.Count, row.Sum)
		if err != nil {
			return err
		}
		rowID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		for _, value := range row.Values {
			if _, err := tx.ExecContext(ctx, `INSERT INTO metric_row_values(row_id, value) VALUES (?, ?)`, rowID, value); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) LoadRows(ctx context.Context, query RowQuery) ([]StoredRow, error) {
	clauses := []string{"1 = 1"}
	args := []any{}
	if query.MetricSet != "" {
		clauses = append(clauses, "metric_set = ?")
		args = append(args, query.MetricSet)
	}
	if query.Metric != "" {
		clauses = append(clauses, "metric = ?")
		args = append(args, query.Metric)
	}
	if !query.Start.IsZero() {
		clauses = append(clauses, "window_start >= ?")
		args = append(args, query.Start.UTC().Format(time.RFC3339Nano))
	}
	if !query.End.IsZero() {
		clauses = append(clauses, "window_start <= ?")
		args = append(args, query.End.UTC().Format(time.RFC3339Nano))
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, metric_set, metric, kind, window_start, labels_json, count, sum FROM metric_rows WHERE `+strings.Join(clauses, " AND ")+` ORDER BY window_start, metric_set, metric`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredRow{}
	for rows.Next() {
		var id int64
		var stored StoredRow
		var windowStart string
		var labelsJSON string
		if err := rows.Scan(&id, &stored.MetricSet, &stored.Row.Metric, &stored.Row.Kind, &windowStart, &labelsJSON, &stored.Row.Count, &stored.Row.Sum); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, windowStart)
		if err != nil {
			return nil, err
		}
		stored.Row.WindowStart = parsed
		if labelsJSON != "" && labelsJSON != "null" {
			if err := json.Unmarshal([]byte(labelsJSON), &stored.Row.Labels); err != nil {
				return nil, err
			}
		}
		values, err := s.loadValues(ctx, id)
		if err != nil {
			return nil, err
		}
		stored.Row.Values = values
		out = append(out, stored)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) loadValues(ctx context.Context, rowID int64) ([]float64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT value FROM metric_row_values WHERE row_id = ? ORDER BY row_id`, rowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []float64{}
	for rows.Next() {
		var value float64
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *SQLiteStore) Metadata(ctx context.Context, metric string) (map[string][]MetadataEntry, error) {
	metric = strings.TrimSpace(metric)
	query := `SELECT metric_set, metric, kind, description, unit, labels_json FROM metric_definitions`
	args := []any{}
	if metric != "" {
		query += ` WHERE metric = ?`
		args = append(args, metric)
	}
	query += ` ORDER BY metric, metric_set`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]MetadataEntry{}
	for rows.Next() {
		var name string
		var labelsJSON string
		var labels []string
		entry := MetadataEntry{}
		if err := rows.Scan(&entry.MetricSet, &name, &entry.Type, &entry.Help, &entry.Unit, &labelsJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
			return nil, err
		}
		entry.Labels = strings.Join(labels, ",")
		out[name] = append(out[name], entry)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

type SQLiteQueryBackend struct {
	store *SQLiteStore
}

func NewSQLiteQueryBackend(store *SQLiteStore) *SQLiteQueryBackend {
	return &SQLiteQueryBackend{store: store}
}

func (b *SQLiteQueryBackend) Capabilities(ctx context.Context) Capabilities {
	return Capabilities{
		Mode:    "local",
		Storage: "sqlite",
		Auth:    false,
		Features: map[string]bool{
			"metadata":   true,
			"query":      true,
			"queryRange": true,
			"sql":        true,
			"writeRows":  true,
		},
		Limits: map[string]any{
			"sqlReadOnly": true,
			"sqlMaxRows":  defaultSQLLimit,
		},
		Schema: defaultSQLSchema(),
	}
}

func (b *SQLiteQueryBackend) Metadata(ctx context.Context, req MetadataRequest) (MetadataResponse, error) {
	metadata, err := b.store.Metadata(ctx, req.Metric)
	if err != nil {
		return MetadataResponse{}, err
	}
	return MetadataResponse{Status: "success", Data: metadata}, nil
}

func (b *SQLiteQueryBackend) Query(ctx context.Context, req QueryRequest) (QueryResponse, error) {
	return NewMemoryQueryBackend(b.store).Query(ctx, req)
}

func (b *SQLiteQueryBackend) QueryRange(ctx context.Context, req QueryRangeRequest) (QueryRangeResponse, error) {
	return NewMemoryQueryBackend(b.store).QueryRange(ctx, req)
}

func (b *SQLiteQueryBackend) SQL(ctx context.Context, req SQLQueryRequest) (SQLQueryResponse, error) {
	query := strings.TrimSpace(req.Query)
	if err := validateReadOnlySQL(query); err != nil {
		return SQLQueryResponse{}, err
	}
	limit := req.Limit
	if limit <= 0 || limit > defaultSQLLimit {
		limit = defaultSQLLimit
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	startedAt := time.Now()
	rows, err := b.store.db.QueryContext(ctx, query, req.Params...)
	if err != nil {
		return SQLQueryResponse{}, err
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return SQLQueryResponse{}, err
	}
	columnTypes, _ := rows.ColumnTypes()
	columns := make([]SQLColumn, len(columnNames))
	for i, name := range columnNames {
		columns[i] = SQLColumn{Name: name}
		if i < len(columnTypes) {
			columns[i].Type = columnTypes[i].DatabaseTypeName()
		}
	}
	outRows := [][]any{}
	for rows.Next() {
		if len(outRows) >= limit {
			break
		}
		values := make([]any, len(columnNames))
		ptrs := make([]any, len(columnNames))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return SQLQueryResponse{}, err
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		outRows = append(outRows, values)
	}
	if err := rows.Err(); err != nil {
		return SQLQueryResponse{}, err
	}
	return SQLQueryResponse{Columns: columns, Rows: outRows, Stats: SQLQueryStats{RowCount: len(outRows), DurationMS: time.Since(startedAt).Milliseconds()}}, nil
}

func validateReadOnlySQL(query string) error {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" {
		return errors.New("SQL query is required")
	}
	if strings.Count(strings.TrimRight(lower, ";"), ";") > 0 {
		return errors.New("multiple SQL statements are not allowed")
	}
	lower = strings.TrimRight(lower, ";")
	if !strings.HasPrefix(lower, "select ") && !strings.HasPrefix(lower, "with ") {
		return errors.New("only read-only SELECT SQL is allowed")
	}
	blocked := []string{" insert ", " update ", " delete ", " drop ", " alter ", " create ", " replace ", " attach ", " detach ", " pragma ", " vacuum "}
	framed := " " + lower + " "
	for _, token := range blocked {
		if strings.Contains(framed, token) {
			return fmt.Errorf("SQL contains blocked token %q", strings.TrimSpace(token))
		}
	}
	return nil
}

func defaultSQLSchema() []SQLSchemaObject {
	schema := []SQLSchemaObject{
		{Name: "metric_sets", Description: "Local metric set names", Columns: []string{"metric_set", "created_at"}},
		{Name: "metric_definitions", Description: "Metric definitions registered by helpers", Columns: []string{"metric_set", "metric", "kind", "description", "unit", "labels_json"}},
		{Name: "metric_rows", Description: "Aggregated metric rows emitted by helpers", Columns: []string{"id", "metric_set", "metric", "kind", "window_start", "labels_json", "count", "sum"}},
		{Name: "metric_row_values", Description: "Histogram sample values for metric rows", Columns: []string{"row_id", "value"}},
	}
	sort.Slice(schema, func(i, j int) bool { return schema[i].Name < schema[j].Name })
	return schema
}
