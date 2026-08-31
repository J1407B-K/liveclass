package global

import (
	"liveclass/internal/rpc/chat/config"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
)

var (
	Config      *config.Config
	RedisClient *redis.Client
	KafkaWriter *kafka.Writer
)
