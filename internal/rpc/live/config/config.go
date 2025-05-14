package config

type Config struct {
	MysqlConfig
	RedisConfig
	GetLiveKeyAddr string
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
