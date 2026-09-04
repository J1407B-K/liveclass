package domain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type BranchMetrics struct {
	Requests *prometheus.CounterVec
	Latency  *prometheus.HistogramVec
	Retries  *prometheus.CounterVec
	handler  http.Handler
}

func NewBranchMetrics(role string) *BranchMetrics {
	registry := prometheus.NewRegistry()
	metrics := &BranchMetrics{
		Requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "mall_saga_branch_total", Help: "DTM Saga branch calls by operation and result.", ConstLabels: prometheus.Labels{"service": role}}, []string{"operation", "result"}),
		Latency:  prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "mall_saga_branch_latency_seconds", Help: "DTM Saga branch latency.", ConstLabels: prometheus.Labels{"service": role}, Buckets: prometheus.DefBuckets}, []string{"operation"}),
		Retries:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "mall_saga_branch_retries_total", Help: "Transient retries inside a DTM branch.", ConstLabels: prometheus.Labels{"service": role}}, []string{"operation", "reason"}),
	}
	registry.MustRegister(metrics.Requests, metrics.Latency, metrics.Retries)
	metrics.handler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	return metrics
}

func (m *BranchMetrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}
	return m.handler
}
