package config

import "time"

type Config struct {
	MongoConfig
	KafkaBroker     string
	KafkaTopic      string
	KafkaGroup      string
	EtcdAddr        string
	JaegerEndpoint  string
	PrometheusPort  string
	ServiceAddr     string
	RedisAddr       string
	RedisPassword   string
	KafkaDispatcher KafkaDispatcherConfig
	FaultInjection  FaultInjectionConfig
}

type FaultInjectionConfig struct {
	MongoDelay time.Duration
}

type KafkaDispatcherConfig struct {
	QueueSize        int
	Workers          int
	EnqueueTimeout   time.Duration
	WriteTimeout     time.Duration
	RetryAttempts    int
	RetryBaseBackoff time.Duration
}

type MongoConfig struct {
	Addr               string
	Database           string
	CollectionPrefix   string
	MessagesCollection string
}
