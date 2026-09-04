package config

import "time"

type Config struct {
	MongoConfig
	KafkaBroker    string
	KafkaTopic     string
	KafkaGroup     string
	EtcdAddr       string
	JaegerEndpoint string
	PrometheusPort string
	ServiceAddr    string
	RedisAddr      string
	RedisPassword  string
	KafkaOutbox    KafkaOutboxConfig
	FaultInjection FaultInjectionConfig
}

type FaultInjectionConfig struct {
	MongoDelay time.Duration
}

type KafkaOutboxConfig struct {
	Workers          int
	PollInterval     time.Duration
	LeaseDuration    time.Duration
	WriteTimeout     time.Duration
	RetryAttempts    int
	RetryBaseBackoff time.Duration
	RetryMaxBackoff  time.Duration
}

type MongoConfig struct {
	Addr               string
	Database           string
	CollectionPrefix   string
	MessagesCollection string
}
