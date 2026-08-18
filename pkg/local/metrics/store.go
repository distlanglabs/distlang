package metrics

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrInvalidMetricSet = errors.New("metric set is required")

type Store interface {
	EnsureMetricSet(ctx context.Context, metricSet string, definitions map[string]Definition) error
	AppendRows(ctx context.Context, metricSet string, rows []Row) error
	LoadRows(ctx context.Context, query RowQuery) ([]StoredRow, error)
	Metadata(ctx context.Context, metric string) (map[string][]MetadataEntry, error)
	Close() error
}

type Definition struct {
	Kind        string   `json:"kind"`
	Description string   `json:"description,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

type Row struct {
	Metric      string            `json:"metric"`
	Kind        string            `json:"kind"`
	WindowStart time.Time         `json:"windowStart"`
	Labels      map[string]string `json:"labels,omitempty"`
	Count       int               `json:"count"`
	Sum         float64           `json:"sum"`
	Values      []float64         `json:"values,omitempty"`
}

type StoredRow struct {
	MetricSet string `json:"metricSet"`
	Row       Row    `json:"row"`
}

type RowQuery struct {
	MetricSet string
	Metric    string
	Start     time.Time
	End       time.Time
}

type MetadataEntry struct {
	Type      string `json:"type"`
	Help      string `json:"help,omitempty"`
	Unit      string `json:"unit,omitempty"`
	MetricSet string `json:"metricSet"`
	Labels    string `json:"labels,omitempty"`
}

type MemoryStore struct {
	mu          sync.RWMutex
	closed      bool
	definitions map[string]map[string]Definition
	rows        map[string][]Row
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		definitions: map[string]map[string]Definition{},
		rows:        map[string][]Row{},
	}
}

func (s *MemoryStore) EnsureMetricSet(ctx context.Context, metricSet string, definitions map[string]Definition) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	metricSet = strings.TrimSpace(metricSet)
	if metricSet == "" {
		return ErrInvalidMetricSet
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("metrics store is closed")
	}
	s.definitions[metricSet] = cloneDefinitions(definitions)
	if _, ok := s.rows[metricSet]; !ok {
		s.rows[metricSet] = []Row{}
	}
	return nil
}

func (s *MemoryStore) AppendRows(ctx context.Context, metricSet string, rows []Row) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	metricSet = strings.TrimSpace(metricSet)
	if metricSet == "" {
		return ErrInvalidMetricSet
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("metrics store is closed")
	}
	for _, row := range rows {
		s.rows[metricSet] = append(s.rows[metricSet], cloneRow(row))
	}
	return nil
}

func (s *MemoryStore) LoadRows(ctx context.Context, query RowQuery) ([]StoredRow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, errors.New("metrics store is closed")
	}

	var out []StoredRow
	for metricSet, rows := range s.rows {
		if query.MetricSet != "" && metricSet != query.MetricSet {
			continue
		}
		for _, row := range rows {
			if query.Metric != "" && row.Metric != query.Metric {
				continue
			}
			if !query.Start.IsZero() && row.WindowStart.Before(query.Start) {
				continue
			}
			if !query.End.IsZero() && row.WindowStart.After(query.End) {
				continue
			}
			out = append(out, StoredRow{MetricSet: metricSet, Row: cloneRow(row)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Row.WindowStart.Equal(out[j].Row.WindowStart) {
			if out[i].MetricSet == out[j].MetricSet {
				return out[i].Row.Metric < out[j].Row.Metric
			}
			return out[i].MetricSet < out[j].MetricSet
		}
		return out[i].Row.WindowStart.Before(out[j].Row.WindowStart)
	})
	return out, nil
}

func (s *MemoryStore) Metadata(ctx context.Context, metric string) (map[string][]MetadataEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, errors.New("metrics store is closed")
	}

	metric = strings.TrimSpace(metric)
	out := map[string][]MetadataEntry{}
	for metricSet, definitions := range s.definitions {
		for name, definition := range definitions {
			if metric != "" && name != metric {
				continue
			}
			out[name] = append(out[name], MetadataEntry{
				Type:      definition.Kind,
				Help:      definition.Description,
				Unit:      definition.Unit,
				MetricSet: metricSet,
				Labels:    strings.Join(definition.Labels, ","),
			})
		}
	}
	return out, nil
}

func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func cloneDefinitions(in map[string]Definition) map[string]Definition {
	out := make(map[string]Definition, len(in))
	for name, definition := range in {
		definition.Labels = append([]string(nil), definition.Labels...)
		out[name] = definition
	}
	return out
}

func cloneRow(in Row) Row {
	out := in
	out.Values = append([]float64(nil), in.Values...)
	if in.Labels != nil {
		out.Labels = make(map[string]string, len(in.Labels))
		for key, value := range in.Labels {
			out.Labels[key] = value
		}
	}
	return out
}
