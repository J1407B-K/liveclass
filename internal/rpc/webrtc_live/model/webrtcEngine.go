package model

import (
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"sync"
)

type Engine struct {
	API                 *webrtc.API
	MediaEngine         *webrtc.MediaEngine
	InterceptorRegistry *interceptor.Registry
	SettingEngine       webrtc.SettingEngine
	SfuConfig           webrtc.Configuration
	BroadcastTracks     sync.Map
	MicTracks           sync.Map
}
