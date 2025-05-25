package global

import (
	"github.com/tencentyun/cos-go-sdk-v5"
	"liveclass/internal/rpc/live/config"
)

var (
	Config *config.Config

	CosClient *cos.Client
)
