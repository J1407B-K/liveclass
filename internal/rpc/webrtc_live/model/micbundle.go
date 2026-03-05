package model

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

type MicBundle struct {
	SessionID string
	Mu        sync.Mutex
	Tracks    []*webrtc.TrackLocalStaticRTP
}
