package model

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

type MicPublisher struct {
	UserID    int64
	SessionID string
	PC        *webrtc.PeerConnection
	Status    ConnectionStatus
	Track     *webrtc.TrackLocalStaticRTP
}

type MicBundle struct {
	Mu         sync.RWMutex
	Publishers map[int64]*MicPublisher
}
