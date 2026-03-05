package config

type Config struct {
	RedisConfig
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
