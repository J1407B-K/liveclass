package config

type Config struct {
	MongoConfig
	KafkaBroker string
	KafkaTopic  string
}

type MongoConfig struct {
	Addr             string
	Database         string
	CollectionPrefix string
}
