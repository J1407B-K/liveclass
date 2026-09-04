package initialize

import (
	"liveclass/internal/rpc/agent/global"
	"os"

	"github.com/spf13/viper"
)

func SetupViper() {
	viper.SetDefault("ChatTemperature", 0.0)
	_ = viper.BindEnv("QdrantConfig.DocCollection", "LIVECLASS_AGENT_DOC_COLLECTION")
	_ = viper.BindEnv("ElasticsearchConfig.DocIndex", "LIVECLASS_AGENT_DOC_INDEX")
	_ = viper.BindEnv("ChatTemperature", "LIVECLASS_AGENT_CHAT_TEMPERATURE")
	_ = viper.BindEnv("MallConfirmSecret", "LIVECLASS_MALL_CONFIRM_SECRET")
	setResilienceDefaults()
	setAgentRuntimeDefaults()
	setRAGDefaults()
	viper.SetDefault("WebSearchURL", "http://openserp:7000/mega/search")
	//先指定文件
	viper.SetConfigType("yaml")
	viper.SetConfigName("agent")
	viper.SetConfigFile(resolveAgentConfigPath())

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

func resolveAgentConfigPath() string {
	if configured := os.Getenv("AGENT_CONFIG_FILE"); configured != "" {
		return configured
	}
	for _, candidate := range []string{"./rpc/manifest/agent.yaml", "./internal/rpc/manifest/agent.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "./rpc/manifest/agent.yaml"
}

func setRAGDefaults() {
	defaults := map[string]any{"ParentSize": 1600, "ChildSize": 400, "Overlap": 80, "ChildTopK": 8, "ParentTopK": 3, "ContextBudgetTokens": 7000}
	for name, value := range defaults {
		viper.SetDefault("RAG."+name, value)
	}
}

func setAgentRuntimeDefaults() {
	defaults := map[string]any{
		"ModelContextTokens": 32768, "SystemReserveTokens": 5000,
		"OutputReserveTokens": 4000, "RAGBudgetTokens": 7000,
		"MemoryBudgetTokens": 3000, "ConversationBudgetTokens": 12000,
		"CompactionTriggerTokens": 10500, "RecentTailTokens": 4000,
		"MaxToolResultTokens": 1200, "MaxSteps": 25, "PlanReminderSteps": 5,
		"PlanMaxSteps": 6, "PlanMaxReplans": 1, "PlanStepMaxReActSteps": 5,
		"PlanExecutionTimeout": "120s",
	}
	for name, value := range defaults {
		viper.SetDefault("AgentRuntime."+name, value)
	}
}

func setResilienceDefaults() {
	defaults := map[string]any{
		"MainLLM.Timeout": "45s", "MainLLM.Attempts": 1,
		"AdvisorLLM.Timeout": "20s", "AdvisorLLM.Attempts": 1,
		"ProfileLLM.Timeout": "8s", "ProfileLLM.Attempts": 1,
		"Embedding.Timeout": "5s", "Embedding.Attempts": 2,
		"Qdrant.Timeout": "800ms", "Qdrant.Attempts": 2,
		"Elasticsearch.Timeout": "800ms", "Elasticsearch.Attempts": 2,
		"Reranker.Timeout": "20s", "Reranker.Attempts": 2,
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
