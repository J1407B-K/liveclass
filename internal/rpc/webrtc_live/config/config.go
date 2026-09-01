package config

type Config struct {
	MysqlConfig
	RedisConfig
	CosConfig
	TmpBaseDir        string
	EtcdAddr          string
	JaegerEndpoint    string
	PrometheusPort    string
	ServiceAddr       string
	NACKEnabled       bool
	PLIMinInterval    string
	RTPDropEveryN     int
	ICEUDPAddr        string
	TrackReadyTimeout string
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
