package initialize

import (
	"liveclass/internal/api/global"

	"github.com/spf13/viper"
)

func SetupViper() {
	viper.SetDefault("ChatWebSocket.SendQueueSize", 256)
	viper.SetDefault("ChatWebSocket.WriteWait", "10s")
	viper.SetDefault("ChatWebSocket.PongWait", "60s")
	viper.SetDefault("ChatWebSocket.PingPeriod", "54s")
	viper.SetDefault("ChatWebSocket.MaxMessageSize", 4096)
	viper.SetDefault("FaultInjection.RedisDelay", "0s")
	viper.SetConfigType("yaml")
	viper.SetConfigName("api")
	viper.SetConfigFile("./manifest/api.yaml")

	err := viper.ReadInConfig()
	if err != nil {
		panic("Read config file failed, err: " + err.Error())
	}

	//数据类型转换
	err = viper.Unmarshal(&global.Config)
	if err != nil {
		panic("Unmarshal config file failed, err: " + err.Error())
	}

}
