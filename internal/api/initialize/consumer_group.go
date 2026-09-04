package initialize

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"liveclass/internal/api/config"
)

// ChatConsumerGroupID gives every API instance its own fanout subscription.
// live_only also changes the group after every reader restart so Kafka backlog
// is never replayed into currently connected rooms. durable_replay retains the
// hostname-stable offset for deployments that explicitly want replay.
func ChatConsumerGroupID(cfg config.ChatKafkaConfig, hostname string) string {
	prefix := strings.Trim(strings.TrimSpace(cfg.GroupPrefix), "-")
	if prefix == "" {
		prefix = "chat-api"
	}
	hostname = strings.Trim(strings.TrimSpace(hostname), "-")
	if hostname == "" {
		hostname = "unknown"
	}
	base := fmt.Sprintf("%s-%s", prefix, hostname)
	if cfg.FanoutMode == "durable_replay" {
		return base
	}
	return base + "-" + uuid.NewString()
}
