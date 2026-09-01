package resilience

import "github.com/prometheus/client_golang/prometheus"

var (
	dependencyRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dependency_requests_total",
		Help: "Logical dependency requests started.",
	}, []string{"dependency", "operation"})
	dependencyErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dependency_errors_total",
		Help: "Logical dependency requests that ended in error.",
	}, []string{"dependency", "operation"})
	dependencyTimeoutTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dependency_timeout_total",
		Help: "Dependency attempts that exhausted their configured timeout.",
	}, []string{"dependency", "operation"})
	dependencyRetryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dependency_retry_total",
		Help: "Dependency retry attempts after the initial attempt.",
	}, []string{"dependency", "operation"})
	dependencyFallbackTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dependency_fallback_total",
		Help: "Dependency failures handled by an explicit fallback.",
	}, []string{"dependency", "operation"})
	dependencyLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dependency_latency_seconds",
		Help:    "End-to-end logical dependency request latency including retries and backoff.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 17),
	}, []string{"dependency", "operation"})
	breakerOpenTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "breaker_open_total",
		Help: "Circuit breaker transitions into the open state.",
	}, []string{"dependency"})
	breakerState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "breaker_state",
		Help: "Circuit breaker state: 0=closed, 1=open, 2=half-open.",
	}, []string{"dependency"})
)

func Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		dependencyRequestsTotal,
		dependencyErrorsTotal,
		dependencyTimeoutTotal,
		dependencyRetryTotal,
		dependencyFallbackTotal,
		dependencyLatency,
		breakerOpenTotal,
		breakerState,
	}
}

func RecordFallback(dependency, operation string) {
	dependencyFallbackTotal.WithLabelValues(dependency, operation).Inc()
}
