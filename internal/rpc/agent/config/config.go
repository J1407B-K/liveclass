package config

type Config struct {
	PostgresConfig
	QdrantConfig
	ElasticsearchConfig
	KafkaBroker    string
	KafkaTopic     string
	APIKey         string
	ChatModel      string
	EmbeddingModel string
	RerankURL      string
	RerankModel    string
	RerankFormat   string
	RedisAddr      string
	EtcdAddr       string
	JaegerEndpoint string
	PrometheusPort string
	ServiceAddr    string
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DB       string
	SSLMode  string
	TimeZone string
}

type QdrantConfig struct {
	Host          string
	GrpcPort      int
	Collection    string
	DocCollection string
	ApiKey        string
}

type ElasticsearchConfig struct {
	Addr     string
	DocIndex string
}
