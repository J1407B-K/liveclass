package main

import (
	"math"
	"sync"
)

// Quantiles are upper bounds rounded to 1 ms. The final bucket is overflow.
type latencyHistogram struct {
	mu      sync.Mutex
	buckets []int64
	count   int64
	maximum float64
}

func newLatencyHistogram() *latencyHistogram {
	return &latencyHistogram{buckets: make([]int64, 600002)}
}

func (h *latencyHistogram) add(ms float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	i := min(max(0, int(math.Ceil(ms))), len(h.buckets)-1)
	h.buckets[i]++
	h.count++
	h.maximum = max(h.maximum, ms)
}

func (h *latencyHistogram) summary() latencySummary {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.count == 0 {
		return latencySummary{}
	}
	quantile := func(p float64) float64 {
		target := int64(math.Ceil(float64(h.count) * p))
		var total int64
		for i, n := range h.buckets {
			total += n
			if total >= target {
				if i == len(h.buckets)-1 {
					return h.maximum
				}
				return float64(i)
			}
		}
		return h.maximum
	}
	return latencySummary{Samples: int(h.count), P50: quantile(.5), P95: quantile(.95), P99: quantile(.99), Max: h.maximum}
}
