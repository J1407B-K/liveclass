package global

import (
	"liveclass/internal/rpc/agent/config"
)

var (
	Config = &config.Config{
		APIKey:         "2971fb58-cb1f-4070-86a9-355a5936bb1a",
		ChatModel:      "ep-20250521012014-nh79b",
		EmbeddingModel: "ep-20250521012113-f6twd",
		RedisAddr:      "127.0.0.1:6380",
	}
)
