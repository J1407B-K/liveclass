package main

import "github.com/prometheus/client_golang/prometheus"

var (
	chatMongoLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_mongo_latency_seconds",
		Help:    "MongoDB insert latency on the LiveChat path.",
		Buckets: prometheus.ExponentialBuckets(0.0005, 2, 16),
	})
	chatPublishLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_publish_latency_seconds",
		Help:    "Kafka WriteMessages latency in Mongo outbox relay workers.",
		Buckets: prometheus.ExponentialBuckets(0.0005, 2, 16),
	})
	chatPublishErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_publish_errors_total",
		Help: "Failed Kafka WriteMessages attempts, including attempts that are later retried.",
	})
	chatOutboxPending = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chat_outbox_pending",
		Help: "Current pending or leased chat outbox records in MongoDB.",
	})
	chatOutboxClaimedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_outbox_claimed_total",
		Help: "MongoDB outbox records claimed for Kafka publishing.",
	})
	chatOutboxPublishedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_outbox_published_total",
		Help: "MongoDB outbox records marked published after Kafka acknowledgement.",
	})
	chatOutboxRetryTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_outbox_retry_total",
		Help: "MongoDB outbox records returned to pending after Kafka failure.",
	})
	chatAcceptedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_accepted_total",
		Help: "Persisted chat messages by realtime delivery status.",
	}, []string{"delivery_status"})
)
