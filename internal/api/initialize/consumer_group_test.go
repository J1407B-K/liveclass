package initialize

import (
	"strings"
	"testing"
	"time"

	"liveclass/internal/api/config"
	"liveclass/internal/api/global"
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

func TestChatKafkaReaderConfigUsesAsyncCommitInterval(t *testing.T) {
	previous := global.Config.ChatKafka
	global.Config.ChatKafka = config.ChatKafkaConfig{
		Broker:         "broker:9092",
		Topic:          "chat",
		CommitInterval: 100 * time.Millisecond,
	}
	defer func() { global.Config.ChatKafka = previous }()

	readerConfig := chatKafkaReaderConfig("chat-api-pod-1")
	if readerConfig.CommitInterval != 100*time.Millisecond {
		t.Fatalf("CommitInterval = %s, want 100ms", readerConfig.CommitInterval)
	}
	if readerConfig.GroupID != "chat-api-pod-1" {
		t.Fatalf("GroupID = %q, want chat-api-pod-1", readerConfig.GroupID)
	}
	if readerConfig.MinBytes != 1 || readerConfig.MaxWait != 50*time.Millisecond {
		t.Fatalf("reader fetch settings changed unexpectedly: min_bytes=%d max_wait=%s", readerConfig.MinBytes, readerConfig.MaxWait)
	}
}
