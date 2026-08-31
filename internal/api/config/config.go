package config

import "time"

type Config struct {
	RedisConfig
	ChatWebSocket  ChatWebSocketConfig
	FaultInjection FaultInjectionConfig
}

type FaultInjectionConfig struct {
	RedisDelay time.Duration
}

type ChatWebSocketConfig struct {
	SendQueueSize  int
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
