package initialize

import (
	"github.com/spf13/viper"
	"liveclass/internal/rpc/quiz/global"
)

func SetupViper() {
	//先指定文件
	viper.SetConfigType("yaml")
	viper.SetConfigName("quiz")
	viper.SetConfigFile("./rpc/manifest/quiz.yaml")

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
