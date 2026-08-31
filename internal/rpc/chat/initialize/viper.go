package initialize

import (
	"github.com/spf13/viper"
	"liveclass/internal/rpc/chat/global"
)

func SetupViper() {
	viper.SetDefault("MongoConfig.MessagesCollection", "messages")
	viper.SetDefault("KafkaDispatcher.QueueSize", 1024)
	viper.SetDefault("KafkaDispatcher.Workers", 4)
	viper.SetDefault("KafkaDispatcher.EnqueueTimeout", "50ms")
	viper.SetDefault("KafkaDispatcher.WriteTimeout", "3s")
	viper.SetDefault("KafkaDispatcher.RetryAttempts", 3)
	viper.SetDefault("KafkaDispatcher.RetryBaseBackoff", "100ms")
	viper.SetDefault("FaultInjection.MongoDelay", "0s")
	//先指定文件
	viper.SetConfigType("yaml")
	viper.SetConfigName("chat")
	viper.SetConfigFile("./rpc/manifest/chat.yaml")

	//读取
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
