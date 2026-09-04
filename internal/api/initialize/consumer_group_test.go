package initialize

import (
	"strings"
	"testing"

	"liveclass/internal/api/config"
)

func TestChatConsumerGroupIDLiveOnlyIsEphemeral(t *testing.T) {
	cfg := config.ChatKafkaConfig{GroupPrefix: "chat-api", FanoutMode: "live_only"}
	first := ChatConsumerGroupID(cfg, "pod-1")
	second := ChatConsumerGroupID(cfg, "pod-1")
	if first == second || !strings.HasPrefix(first, "chat-api-pod-1-") {
		t.Fatalf("live-only group IDs must be unique per reader: %q %q", first, second)
	}
}

func TestChatConsumerGroupIDDurableReplayIsStable(t *testing.T) {
	cfg := config.ChatKafkaConfig{GroupPrefix: "chat-api", FanoutMode: "durable_replay"}
	if got := ChatConsumerGroupID(cfg, "pod-1"); got != "chat-api-pod-1" {
		t.Fatalf("durable group ID = %q", got)
	}
}
