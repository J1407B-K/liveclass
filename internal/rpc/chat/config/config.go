package config

type Config struct {
	MongoConfig
}

type MongoConfig struct {
	Addr       string
	Database   string
	Collection string
}
