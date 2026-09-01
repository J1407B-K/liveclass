package initialize

import (
	"liveclass/internal/rpc/agent/global"

	"github.com/spf13/viper"
)

func SetupViper() {
	setResilienceDefaults()
	viper.SetDefault("WebSearchURL", "http://openserp:7000/mega/search")
	//先指定文件
	viper.SetConfigType("yaml")
	viper.SetConfigName("agent")
	viper.SetConfigFile("./rpc/manifest/agent.yaml")

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

func setResilienceDefaults() {
	defaults := map[string]any{
		"MainLLM.Timeout": "45s", "MainLLM.Attempts": 1,
		"AdvisorLLM.Timeout": "3s", "AdvisorLLM.Attempts": 1,
		"ProfileLLM.Timeout": "8s", "ProfileLLM.Attempts": 1,
		"Embedding.Timeout": "5s", "Embedding.Attempts": 2,
		"Qdrant.Timeout": "800ms", "Qdrant.Attempts": 2,
		"Elasticsearch.Timeout": "800ms", "Elasticsearch.Attempts": 2,
		"Reranker.Timeout": "2s", "Reranker.Attempts": 2,
		"WebSearch.Timeout": "2s", "WebSearch.Attempts": 2,
		"PostgresRead.Timeout": "800ms", "PostgresRead.Attempts": 2,
		"PostgresWrite.Timeout": "1500ms", "PostgresWrite.Attempts": 1,
		"InternalRPC.Timeout": "800ms", "InternalRPC.Attempts": 2,
	}
	for name, value := range defaults {
		prefix := "Resilience." + name
		viper.SetDefault(prefix, value)
	}
	for _, name := range []string{"Embedding", "Qdrant", "Elasticsearch", "Reranker", "WebSearch", "AdvisorLLM", "ProfileLLM", "MainLLM"} {
		prefix := "Resilience." + name + ".Breaker."
		viper.SetDefault(prefix+"Enabled", true)
		viper.SetDefault(prefix+"RollingWindow", "30s")
		viper.SetDefault(prefix+"MinimumRequests", 10)
		viper.SetDefault(prefix+"FailureThreshold", 0.5)
		viper.SetDefault(prefix+"OpenDuration", "10s")
		viper.SetDefault(prefix+"HalfOpenProbes", 2)
	}
	for _, name := range []string{"Embedding", "Qdrant", "Elasticsearch", "Reranker", "WebSearch", "PostgresRead", "InternalRPC"} {
		prefix := "Resilience." + name + "."
		viper.SetDefault(prefix+"Backoff", "50ms")
		viper.SetDefault(prefix+"MaxBackoff", "200ms")
	}
}
