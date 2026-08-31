package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	Registry = prometheus.NewRegistry()

	ActiveWebSocketConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "active_websocket_connections",
		Help: "Current number of active chat WebSocket connections.",
	})
	RoomConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "room_connections",
		Help: "Current chat WebSocket connections by lesson room.",
	}, []string{"lesson_id"})
	ChatQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chat_queue_depth",
		Help: "Current total number of messages waiting in client send queues.",
	})
	SlowConsumerTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "slow_consumer_total",
		Help: "Clients evicted because their bounded send queue was full.",
	})
	DroppedMessagesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dropped_messages_total",
		Help: "Fanout deliveries dropped because a client send queue was full.",
	})
	SubscriberReconnectTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "subscriber_reconnect_total",
		Help: "Kafka chat consumer recreation attempts after fetch failures.",
	})
	SubscriberConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chat_subscriber_connected",
		Help: "Whether the API Kafka chat consumer is currently fetching.",
	})
	ChatMessagesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_total",
		Help: "Chat messages accepted from WebSocket clients.",
	})
	ChatRedisRateLimitLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_redis_rate_limit_latency_seconds",
		Help:    "Redis Lua rate-limit latency on the chat send path.",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 16),
	})
	ChatRPCLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_rpc_latency_seconds",
		Help:    "Kitex Chat.LiveChat latency observed by the API.",
		Buckets: prometheus.ExponentialBuckets(0.0005, 2, 16),
	})
	ChatFanoutLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_fanout_latency_seconds",
		Help:    "Duration of one in-process room fanout into bounded client queues.",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 16),
	})
	WebSocketWriteErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "websocket_write_errors_total",
		Help: "Chat WebSocket write failures.",
	})
)

func init() {
	Registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		ActiveWebSocketConnections,
		RoomConnections,
		ChatQueueDepth,
		SlowConsumerTotal,
		DroppedMessagesTotal,
		SubscriberReconnectTotal,
		SubscriberConnected,
		ChatMessagesTotal,
		ChatRedisRateLimitLatency,
		ChatRPCLatency,
		ChatFanoutLatency,
		WebSocketWriteErrorsTotal,
	)
}
