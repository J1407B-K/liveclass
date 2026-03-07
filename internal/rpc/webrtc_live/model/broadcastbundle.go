package model

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

type ConnectionStatus int8

const (
	ConnectionConnecting ConnectionStatus = iota
	ConnectionConnected
	ConnectionDisconnected
	ConnectionFailed
	ConnectionClosed
)

type BroadcastBundle struct {
	SessionID string
	Mu        sync.RWMutex

	PublisherPC     *webrtc.PeerConnection
	PublisherStatus ConnectionStatus

	VideoTrack *webrtc.TrackLocalStaticRTP
	AudioTrack *webrtc.TrackLocalStaticRTP
}
