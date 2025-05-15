package config

type Config struct {
	MysqlConfig
	RedisConfig
	GetLiveKeyAddr string
	RTMPPlayAddr   string
	FLVPlayAddr    string
	HLSPlayAddr    string
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
