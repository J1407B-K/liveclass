package initialize

import (
	"liveclass/internal/api/global"

	"github.com/spf13/viper"
)

func SetupViper() {
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
