package initialize

import (
	"github.com/spf13/viper"
	"liveclass/internal/rpc/chat/global"
)

func SetupViper() {
	_ = viper.BindEnv("KafkaBroker", "LIVECLASS_CHAT_KAFKA_BROKER")
	_ = viper.BindEnv("KafkaTopic", "LIVECLASS_CHAT_KAFKA_TOPIC")
	viper.SetDefault("MongoConfig.MessagesCollection", "messages")
	viper.SetDefault("KafkaOutbox.Workers", 4)
	viper.SetDefault("KafkaOutbox.PollInterval", "200ms")
	viper.SetDefault("KafkaOutbox.LeaseDuration", "15s")
	viper.SetDefault("KafkaOutbox.WriteTimeout", "3s")
	viper.SetDefault("KafkaOutbox.RetryAttempts", 2)
	viper.SetDefault("KafkaOutbox.RetryBaseBackoff", "100ms")
	viper.SetDefault("KafkaOutbox.RetryMaxBackoff", "30s")
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
	if global.Config.KafkaBroker == "" || global.Config.KafkaTopic == "" {
		panic("KafkaBroker and KafkaTopic are required")
	}

}
