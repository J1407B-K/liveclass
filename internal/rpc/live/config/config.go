package config

type Config struct {
	MysqlConfig
	RedisConfig
	CosConfig
	GetLiveKeyAddr string
	RTMPPlayAddr   string
	FLVPlayAddr    string
	HLSPlayAddr    string
	TmpBaseDir     string
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

type CosConfig struct {
	SecretId        string
	SecretKey       string
	BucketnameAppid string
	CosRegion       string
}
