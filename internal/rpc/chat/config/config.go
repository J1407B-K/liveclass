package config

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
}

type MongoConfig struct {
	Addr             string
	Database         string
	CollectionPrefix string
}
