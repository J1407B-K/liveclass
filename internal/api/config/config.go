package config

import "time"

type Config struct {
	RedisConfig
	ChatWebSocket     ChatWebSocketConfig
	WebSocketSecurity WebSocketSecurityConfig
	ChatKafka         ChatKafkaConfig
	FaultInjection    FaultInjectionConfig
}

type WebSocketSecurityConfig struct {
	AllowedOrigins  []string
	AllowQueryToken bool
	SecureCookies   bool
}

type ChatKafkaConfig struct {
	Broker      string
	Topic       string
	GroupPrefix string
	FanoutMode  string
}

type FaultInjectionConfig struct {
	RedisDelay time.Duration
}

type ChatWebSocketConfig struct {
	SendQueueSize    int
	MessageDedupSize int
	WriteWait        time.Duration
	PongWait         time.Duration
	PingPeriod       time.Duration
	MaxMessageSize   int64
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
