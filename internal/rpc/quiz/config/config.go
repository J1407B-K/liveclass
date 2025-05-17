package config

type Config struct {
	MysqlConfig
}

type MysqlConfig struct {
	Username string
	Password string
	Addr     string
	DB       string
}
