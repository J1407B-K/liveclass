package model

import (
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"sync"
)

type Engine struct {
	MediaEngine         *webrtc.MediaEngine
	InterceptorRegistry *interceptor.Registry
	SfuConfig           webrtc.Configuration
	BroadcastTracks     sync.Map //map[string]*webrtc.TrackLocalStaticRTP
	DuplexConns         sync.Map //map[string][]*webrtc.PeerConnection
}
