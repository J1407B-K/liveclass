package config

import "time"

type Config struct {
	PostgresConfig
	QdrantConfig
	ElasticsearchConfig
	KafkaBroker       string
	KafkaTopic        string
	APIKey            string
	ChatModel         string
	ChatTemperature   float32
	EmbeddingModel    string
	RerankURL         string
	RerankModel       string
	RerankFormat      string
	WebSearchURL      string
	RedisAddr         string
	EtcdAddr          string
	JaegerEndpoint    string
	PrometheusPort    string
	ServiceAddr       string
	MallConfirmSecret string
	AgentRuntime      AgentRuntimeConfig
	RAG               RAGConfig
	Resilience        ResilienceConfig
}

type RAGConfig struct {
	ParentSize          int
	ChildSize           int
	Overlap             int
	ChildTopK           int
	ParentTopK          int
	ContextBudgetTokens int
}

type AgentRuntimeConfig struct {
	ModelContextTokens       int
	SystemReserveTokens      int
	OutputReserveTokens      int
	RAGBudgetTokens          int
	MemoryBudgetTokens       int
	ConversationBudgetTokens int
	CompactionTriggerTokens  int
	RecentTailTokens         int
	MaxToolResultTokens      int
	MaxSteps                 int
	PlanReminderSteps        int
	PlanMaxSteps             int
	PlanMaxReplans           int
	PlanStepMaxReActSteps    int
	PlanExecutionTimeout     time.Duration
}

type ResilienceConfig struct {
	MainLLM       DependencyPolicyConfig
	AdvisorLLM    DependencyPolicyConfig
	ProfileLLM    DependencyPolicyConfig
	Embedding     DependencyPolicyConfig
	Qdrant        DependencyPolicyConfig
	Elasticsearch DependencyPolicyConfig
	Reranker      DependencyPolicyConfig
	WebSearch     DependencyPolicyConfig
	PostgresRead  DependencyPolicyConfig
	PostgresWrite DependencyPolicyConfig
	InternalRPC   DependencyPolicyConfig
}

type DependencyPolicyConfig struct {
	Timeout    time.Duration
	Attempts   int
	Backoff    time.Duration
	MaxBackoff time.Duration
	Breaker    BreakerConfig
}

type BreakerConfig struct {
	Enabled          bool
	RollingWindow    time.Duration
	MinimumRequests  int
	FailureThreshold float64
	OpenDuration     time.Duration
	HalfOpenProbes   int
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DB       string
	SSLMode  string
	TimeZone string
}

type QdrantConfig struct {
	Host          string
	GrpcPort      int
	Collection    string
	DocCollection string
	ApiKey        string
}

type ElasticsearchConfig struct {
	Addr     string
	DocIndex string
}
