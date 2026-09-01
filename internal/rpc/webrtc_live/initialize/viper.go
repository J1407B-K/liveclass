package initialize

import (
	"github.com/spf13/viper"
	"liveclass/internal/rpc/webrtc_live/global"
)

func SetupViper() {
	//先指定文件
	viper.SetConfigType("yaml")
	viper.SetConfigName("webrtc_live")
	viper.SetConfigFile("./rpc/manifest/webrtc_live.yaml")
	viper.SetDefault("NACKEnabled", true)
	viper.SetDefault("PLIMinInterval", "500ms")
	viper.SetDefault("RTPDropEveryN", 0)
	viper.SetDefault("ICEUDPAddr", ":50000")
	viper.SetDefault("TrackReadyTimeout", "3s")
	_ = viper.BindEnv("NACKEnabled", "LIVECLASS_WEBRTC_NACK_ENABLED")
	_ = viper.BindEnv("PLIMinInterval", "LIVECLASS_WEBRTC_PLI_MIN_INTERVAL")
	_ = viper.BindEnv("RTPDropEveryN", "LIVECLASS_WEBRTC_RTP_DROP_EVERY_N")
	_ = viper.BindEnv("ICEUDPAddr", "LIVECLASS_WEBRTC_ICE_UDP_ADDR")
	_ = viper.BindEnv("TrackReadyTimeout", "LIVECLASS_WEBRTC_TRACK_READY_TIMEOUT")

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
