package config

type Config struct {
	MysqlConfig
	RedisConfig
	EtcdAddr       string
	JaegerEndpoint string
	PrometheusPort string
	ServiceAddr    string
}

type MysqlConfig struct {
	Username string
	Password string
	Addr     string
	DB       string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
