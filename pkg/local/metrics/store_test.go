package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreMetadata(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	err := store.EnsureMetricSet(ctx, "quickstart-app", map[string]Definition{
		"requestCount": {Kind: "counter", Description: "Requests", Unit: "1", Labels: []string{"route"}},
		"latencyMs":    {Kind: "histogram", Description: "Latency", Unit: "ms"},
	})
	if err != nil {
		t.Fatalf("EnsureMetricSet: %v", err)
	}

	metadata, err := store.Metadata(ctx, "requestCount")
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	entries := metadata["requestCount"]
	if len(entries) != 1 {
		t.Fatalf("expected one requestCount metadata entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Type != "counter" || entry.Help != "Requests" || entry.Unit != "1" || entry.MetricSet != "quickstart-app" || entry.Labels != "route" {
		t.Fatalf("unexpected metadata entry: %#v", entry)
	}
	if _, ok := metadata["latencyMs"]; ok {
		t.Fatalf("metric filter returned latencyMs metadata")
	}
}

func TestMemoryStoreAppendAndLoadRows(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	start := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	if err := store.AppendRows(ctx, "quickstart-app", []Row{
		{Metric: "requestCount", Kind: "counter", WindowStart: start.Add(time.Minute), Labels: map[string]string{"route": "/"}, Count: 2, Sum: 2},
		{Metric: "latencyMs", Kind: "histogram", WindowStart: start.Add(2 * time.Minute), Count: 2, Sum: 75, Values: []float64{25, 50}},
		{Metric: "requestCount", Kind: "counter", WindowStart: start.Add(3 * time.Minute), Labels: map[string]string{"route": "/health"}, Count: 1, Sum: 1},
	}); err != nil {
		t.Fatalf("AppendRows: %v", err)
	}
	if err := store.AppendRows(ctx, "other-app", []Row{{Metric: "requestCount", Kind: "counter", WindowStart: start, Count: 1, Sum: 1}}); err != nil {
		t.Fatalf("AppendRows other-app: %v", err)
	}

	rows, err := store.LoadRows(ctx, RowQuery{
		MetricSet: "quickstart-app",
		Metric:    "requestCount",
		Start:     start.Add(30 * time.Second),
		End:       start.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("LoadRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one matching row, got %d", len(rows))
	}
	row := rows[0]
	if row.MetricSet != "quickstart-app" || row.Row.Metric != "requestCount" || row.Row.Sum != 2 || row.Row.Labels["route"] != "/" {
		t.Fatalf("unexpected row: %#v", row)
	}

	row.Row.Labels["route"] = "mutated"
	loadedAgain, err := store.LoadRows(ctx, RowQuery{MetricSet: "quickstart-app", Metric: "requestCount"})
	if err != nil {
		t.Fatalf("LoadRows again: %v", err)
	}
	if loadedAgain[0].Row.Labels["route"] != "/" {
		t.Fatalf("store returned mutable row labels")
	}
}

func TestMemoryStoreRejectsEmptyMetricSet(t *testing.T) {
	store := NewMemoryStore()
	err := store.EnsureMetricSet(context.Background(), " ", nil)
	if !errors.Is(err, ErrInvalidMetricSet) {
		t.Fatalf("expected ErrInvalidMetricSet, got %v", err)
	}
}
