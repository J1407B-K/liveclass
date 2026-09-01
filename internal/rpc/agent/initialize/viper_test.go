package initialize

import (
	"testing"
	"time"

	"liveclass/internal/rpc/agent/config"

	"github.com/spf13/viper"
)

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
