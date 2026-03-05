package model

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

type BroadcastBundle struct {
	SessionID string
	Mu        sync.Mutex
	Tracks    []*webrtc.TrackLocalStaticRTP
}
