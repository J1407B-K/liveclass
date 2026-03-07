package global

import (
	"liveclass/internal/rpc/chat/config"

	"github.com/segmentio/kafka-go"
)

var (
	Config *config.Config
	Writer *kafka.Writer
)
