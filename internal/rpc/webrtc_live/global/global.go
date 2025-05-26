package global

import (
	"liveclass/internal/rpc/webrtc_live/config"
	"liveclass/internal/rpc/webrtc_live/model"
)

var (
	Config       *config.Config
	WebRTCEngine *model.Engine
)
