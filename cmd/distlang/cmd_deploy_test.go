package main

import (
	"reflect"
	"testing"
)

func TestInferMetricsBuckets(t *testing.T) {
	tests := []struct {
		name    string
		emitted string
		want    []string
	}{
		{
			name:    "finds unique sorted buckets",
			emitted: `const a = instantiateMetrics({ reqs: "counter" }, "z-bucket"); const b = instantiateMetrics({ reqs: "counter" }, "a-bucket"); const c = instantiateMetrics({ reqs: "counter" }, "z-bucket");`,
			want:    []string{"a-bucket", "z-bucket"},
		},
		{
			name:    "ignores dynamic buckets",
			emitted: `const a = instantiateMetrics({ reqs: "counter" }, bucketName);`,
			want:    nil,
		},
		{
			name:    "supports single quotes",
			emitted: `const a = instantiateMetrics({ reqs: "counter" }, 'echo-metrics');`,
			want:    []string{"echo-metrics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferMetricsBuckets(tt.emitted)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("inferMetricsBuckets() = %v, want %v", got, tt.want)
			}
		})
	}
}
