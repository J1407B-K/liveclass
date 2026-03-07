package model

import (
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

type Engine struct {
	API                 *webrtc.API
	MediaEngine         *webrtc.MediaEngine
	InterceptorRegistry *interceptor.Registry
	SettingEngine       webrtc.SettingEngine
	SfuConfig           webrtc.Configuration
	BroadcastRooms      sync.Map
	MicTracks           sync.Map
}
