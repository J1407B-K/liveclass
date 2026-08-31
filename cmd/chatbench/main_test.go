package main

import (
	"strings"
	"testing"
	"time"
)

func TestMarkerTimestamp(t *testing.T) {
	want := time.Unix(0, 123456789)
	got, ok := markerTimestamp("chatbench run1 7 123456789", "run1")
	if !ok || !got.Equal(want) {
		t.Fatalf("got %v, %v; want %v, true", got, ok, want)
	}
	if _, ok := markerTimestamp("chatbench another 7 123456789", "run1"); ok {
		t.Fatal("accepted another run")
	}
}

func TestSummarize(t *testing.T) {
	got := summarize([]float64{100, 1, 50, 99, 10})
	if got.Samples != 5 || got.P50 != 50 || got.P95 != 99 || got.P99 != 99 || got.Max != 100 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestParseMetrics(t *testing.T) {
	input := `# HELP go_goroutines Number of goroutines.
go_goroutines 42
go_memstats_heap_alloc_bytes 1024
ignored_metric 7
request_total{method="GET"} 10
`
	got, err := parseMetrics(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["go_goroutines"] != 42 || got["go_memstats_heap_alloc_bytes"] != 1024 {
		t.Fatalf("unexpected metrics: %#v", got)
	}
}
