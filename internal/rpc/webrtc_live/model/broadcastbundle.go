package model

import (
	"sync"
	"time"

	"github.com/pion/rtcp"
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
	ExpectsAudio    bool

	VideoTrack *webrtc.TrackLocalStaticRTP
	AudioTrack *webrtc.TrackLocalStaticRTP
	VideoSSRC  uint32
	lastPLI    time.Time
}

// RequestVideoKeyframe collapses many viewer requests into one upstream PLI per interval.
func (b *BroadcastBundle) RequestVideoKeyframe(now time.Time, minInterval time.Duration) (bool, error) {
	b.Mu.Lock()
	if b.PublisherPC == nil || b.VideoSSRC == 0 || now.Sub(b.lastPLI) < minInterval {
		b.Mu.Unlock()
		return false, nil
	}
	b.lastPLI = now
	pc, ssrc := b.PublisherPC, b.VideoSSRC
	b.Mu.Unlock()

	if err := pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: ssrc}}); err != nil {
		return false, err
	}
	return true, nil
}
