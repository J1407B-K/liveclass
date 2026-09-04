package initialize

import (
	"liveclass/internal/api/global"

	"github.com/spf13/viper"
)

func SetupViper() {
	_ = viper.BindEnv("ChatKafka.Broker", "LIVECLASS_CHAT_KAFKA_BROKER")
	_ = viper.BindEnv("ChatKafka.Topic", "LIVECLASS_CHAT_KAFKA_TOPIC")
	_ = viper.BindEnv("ChatKafka.FanoutMode", "LIVECLASS_CHAT_FANOUT_MODE")
	_ = viper.BindEnv("WebSocketSecurity.AllowQueryToken", "LIVECLASS_WS_ALLOW_QUERY_TOKEN")
	_ = viper.BindEnv("WebSocketSecurity.SecureCookies", "LIVECLASS_WS_SECURE_COOKIES")
	viper.SetDefault("ChatWebSocket.SendQueueSize", 256)
	viper.SetDefault("ChatWebSocket.MessageDedupSize", 1024)
	viper.SetDefault("ChatWebSocket.WriteWait", "10s")
	viper.SetDefault("ChatWebSocket.PongWait", "60s")
	viper.SetDefault("ChatWebSocket.PingPeriod", "54s")
	viper.SetDefault("ChatWebSocket.MaxMessageSize", 4096)
	viper.SetDefault("WebSocketSecurity.AllowQueryToken", false)
	viper.SetDefault("WebSocketSecurity.SecureCookies", false)
	viper.SetDefault("ChatKafka.Broker", "127.0.0.1:9092")
	viper.SetDefault("ChatKafka.Topic", "liveclass-chat")
	viper.SetDefault("ChatKafka.GroupPrefix", "chat-api")
	viper.SetDefault("ChatKafka.FanoutMode", "durable_replay")
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
	if global.Config.ChatKafka.FanoutMode != "live_only" && global.Config.ChatKafka.FanoutMode != "durable_replay" {
		panic("ChatKafka.FanoutMode must be live_only or durable_replay")
	}
	if global.Config.ChatKafka.Broker == "" || global.Config.ChatKafka.Topic == "" {
		panic("ChatKafka.Broker and ChatKafka.Topic are required")
	}
	global.ConfigureWebSocketUpgrader(global.Config.WebSocketSecurity.AllowedOrigins)

}
