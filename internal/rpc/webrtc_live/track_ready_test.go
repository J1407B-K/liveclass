package main

import (
	"context"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"

	"liveclass/internal/rpc/webrtc_live/model"
)

func TestOfferExpectsSendingMedia(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		want      bool
	}{
		{name: "sendrecv", direction: "sendrecv", want: true},
		{name: "sendonly", direction: "sendonly", want: true},
		{name: "recvonly", direction: "recvonly", want: false},
		{name: "inactive", direction: "inactive", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\nm=audio 9 UDP/TLS/RTP/SAVPF 111\r\na=" + tt.direction + "\r\n"}
			if got := offerExpectsSendingMedia(offer, "audio"); got != tt.want {
				t.Fatalf("offerExpectsSendingMedia()=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestWaitForBroadcastTracksWaitsForExpectedAudio(t *testing.T) {
	video, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "video", "stream")
	if err != nil {
		t.Fatal(err)
	}
	audio, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2}, "audio", "stream")
	if err != nil {
		t.Fatal(err)
	}
	b := &model.BroadcastBundle{PublisherStatus: model.ConnectionConnected, ExpectsAudio: true, VideoTrack: video}
	go func() {
		time.Sleep(30 * time.Millisecond)
		b.Mu.Lock()
		b.AudioTrack = audio
		b.Mu.Unlock()
	}()

	gotVideo, gotAudio, err := waitForBroadcastTracks(context.Background(), b, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if gotVideo != video || gotAudio != audio {
		t.Fatal("waitForBroadcastTracks returned before both promised tracks were ready")
	}
}

func TestWaitForBroadcastTracksAllowsVideoOnlyPublisher(t *testing.T) {
	video, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "video", "stream")
	if err != nil {
		t.Fatal(err)
	}
	b := &model.BroadcastBundle{PublisherStatus: model.ConnectionConnected, VideoTrack: video}
	gotVideo, gotAudio, err := waitForBroadcastTracks(context.Background(), b, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if gotVideo != video || gotAudio != nil {
		t.Fatal("video-only publisher should not wait for an unoffered audio track")
	}
}
