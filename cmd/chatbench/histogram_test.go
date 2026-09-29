package main

import "testing"

func TestHistogramQuantilesAndOverflow(t *testing.T) {
	h := newLatencyHistogram()
	if h.summary().Samples != 0 {
		t.Fatal("nonempty histogram")
	}
	for i := 1; i <= 100; i++ {
		h.add(float64(i) - .1)
	}
	s := h.summary()
	if s.Samples != 100 || s.P50 != 50 || s.P95 != 95 || s.P99 != 99 || s.Max != 99.9 {
		t.Fatalf("unexpected: %+v", s)
	}
	h = newLatencyHistogram()
	h.add(700000)
	if h.summary().P99 != 700000 {
		t.Fatal("overflow truncated")
	}
}
