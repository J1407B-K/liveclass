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
		Help:    "Kafka WriteMessages latency in bounded dispatcher workers.",
		Buckets: prometheus.ExponentialBuckets(0.0005, 2, 16),
	})
	chatPublishErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_publish_errors_total",
		Help: "Failed Kafka WriteMessages attempts, including attempts that are later retried.",
	})
	chatPublishQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chat_publish_queue_depth",
		Help: "Current messages waiting in bounded Kafka dispatcher queues.",
	})
	chatPublishQueueFullTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_publish_queue_full_total",
		Help: "Messages rejected after the Kafka dispatcher enqueue timeout.",
	})
)
