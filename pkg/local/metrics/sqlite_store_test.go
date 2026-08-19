package metrics

import (
	"context"
	"testing"
	"time"
)

func TestSQLiteStorePersistsRowsAndRunsSQL(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/metrics.db"
	store, err := OpenSQLiteStore(path)
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}
	ts := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	if err := store.EnsureMetricSet(ctx, "local-app", map[string]Definition{"requests": {Kind: "counter", Description: "Requests", Unit: "count", Labels: []string{"route"}}}); err != nil {
		t.Fatalf("EnsureMetricSet: %v", err)
	}
	if err := store.AppendRows(ctx, "local-app", []Row{{Metric: "requests", Kind: "counter", WindowStart: ts, Labels: map[string]string{"route": "/"}, Count: 2, Sum: 2}}); err != nil {
		t.Fatalf("AppendRows: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	store, err = OpenSQLiteStore(path)
	if err != nil {
		t.Fatalf("reopen SQLiteStore: %v", err)
	}
	defer store.Close()
	metadata, err := store.Metadata(ctx, "requests")
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if len(metadata["requests"]) != 1 || metadata["requests"][0].MetricSet != "local-app" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	rows, err := store.LoadRows(ctx, RowQuery{MetricSet: "local-app", Metric: "requests"})
	if err != nil {
		t.Fatalf("LoadRows: %v", err)
	}
	if len(rows) != 1 || rows[0].Row.Sum != 2 || rows[0].Row.Labels["route"] != "/" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	backend := NewSQLiteQueryBackend(store)
	sqlResult, err := backend.SQL(ctx, SQLQueryRequest{Query: `select metric_set, metric, sum from metric_rows where metric_set = 'local-app'`, Limit: 10})
	if err != nil {
		t.Fatalf("SQL: %v", err)
	}
	if len(sqlResult.Rows) != 1 || sqlResult.Rows[0][0] != "local-app" || sqlResult.Rows[0][1] != "requests" {
		t.Fatalf("unexpected SQL result: %#v", sqlResult)
	}
}

func TestSQLiteStoreCapabilitiesEnableSQL(t *testing.T) {
	store, err := OpenSQLiteStore(t.TempDir() + "/metrics.db")
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}
	defer store.Close()
	capabilities := NewSQLiteQueryBackend(store).Capabilities(context.Background())
	if capabilities.Storage != "sqlite" || !capabilities.Features["sql"] || capabilities.Auth {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
}

func TestSQLiteStoreSQLRejectsWrites(t *testing.T) {
	store, err := OpenSQLiteStore(t.TempDir() + "/metrics.db")
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}
	defer store.Close()
	if _, err := NewSQLiteQueryBackend(store).SQL(context.Background(), SQLQueryRequest{Query: `delete from metric_rows`}); err == nil {
		t.Fatalf("expected write SQL to be rejected")
	}
}
