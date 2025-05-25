package initialize

import (
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/webrtc/v4"
	"liveclass/internal/rpc/webrtc_live/global"
	"liveclass/internal/rpc/webrtc_live/model"
)

func InitWebRTCEngine() {
	mediaEngine := &webrtc.MediaEngine{}
	mediaEngine.RegisterDefaultCodecs()

	interceptorRegistry := &interceptor.Registry{}
	webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry)
	if pli, err := intervalpli.NewReceiverInterceptor(); err == nil {
		interceptorRegistry.Add(pli)
	}

	sfuConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"turn:your-turn-server:5349"}, Username: "user", Credential: "pass"},
		},
	}

	global.WebRTCEngine = &model.Engine{
		MediaEngine:         mediaEngine,
		InterceptorRegistry: interceptorRegistry,
		SfuConfig:           sfuConfig,
	}
}
