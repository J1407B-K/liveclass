package initialize

import (
	"os"
	"testing"
	"time"

	"liveclass/internal/rpc/agent/config"

	"github.com/spf13/viper"
)

func TestResolveAgentConfigPathSupportsRepositoryRoot(t *testing.T) {
	old, had := os.LookupEnv("AGENT_CONFIG_FILE")
	_ = os.Unsetenv("AGENT_CONFIG_FILE")
	defer func() {
		if had {
			_ = os.Setenv("AGENT_CONFIG_FILE", old)
		} else {
			_ = os.Unsetenv("AGENT_CONFIG_FILE")
		}
	}()
	path := resolveAgentConfigPath()
	if path != "./internal/rpc/manifest/agent.yaml" && path != "./rpc/manifest/agent.yaml" {
		t.Fatalf("unexpected config path: %s", path)
	}
}

func TestResilienceDefaultsUnmarshal(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setResilienceDefaults()

	var got struct {
		Resilience config.ResilienceConfig
	}
	if err := viper.Unmarshal(&got); err != nil {
		t.Fatal(err)
	}
	if got.Resilience.MainLLM.Timeout != 45*time.Second || got.Resilience.MainLLM.Attempts != 1 {
		t.Fatalf("main LLM policy=%+v", got.Resilience.MainLLM)
	}
	if got.Resilience.PostgresRead.Timeout != 800*time.Millisecond || got.Resilience.PostgresRead.Attempts != 2 {
		t.Fatalf("postgres read policy=%+v", got.Resilience.PostgresRead)
	}
	if got.Resilience.PostgresWrite.Timeout != 1500*time.Millisecond || got.Resilience.PostgresWrite.Attempts != 1 {
		t.Fatalf("postgres write policy=%+v", got.Resilience.PostgresWrite)
	}
	if !got.Resilience.Qdrant.Breaker.Enabled || got.Resilience.Qdrant.Breaker.HalfOpenProbes != 2 {
		t.Fatalf("qdrant breaker=%+v", got.Resilience.Qdrant.Breaker)
	}
}

func TestAgentRuntimePlanningDefaultsUnmarshal(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setAgentRuntimeDefaults()

	var got struct {
		AgentRuntime config.AgentRuntimeConfig
	}
	if err := viper.Unmarshal(&got); err != nil {
		t.Fatal(err)
	}
	if got.AgentRuntime.PlanMaxSteps != 6 || got.AgentRuntime.PlanMaxReplans != 1 || got.AgentRuntime.PlanStepMaxReActSteps != 5 {
		t.Fatalf("planning limits=%+v", got.AgentRuntime)
	}
	if got.AgentRuntime.PlanExecutionTimeout != 120*time.Second {
		t.Fatalf("plan execution timeout=%s", got.AgentRuntime.PlanExecutionTimeout)
	}
}
