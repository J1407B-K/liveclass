package initialize

import (
	"testing"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

type countWriter struct{ writes int }

func (w *countWriter) Write(*rtp.Header, []byte, interceptor.Attributes) (int, error) {
	w.writes++
	return 1, nil
}

func TestRTPDropInterceptorDropsEveryNthVideoPacket(t *testing.T) {
	drops := 0
	i := &rtpDropInterceptor{everyN: 3, onDrop: func() { drops++ }}
	underlying := &countWriter{}
	w := i.BindLocalStream(&interceptor.StreamInfo{MimeType: "video/VP8"}, underlying)
	for range 7 {
		if _, err := w.Write(&rtp.Header{}, []byte{1}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if drops != 2 || underlying.writes != 5 {
		t.Fatalf("drops=%d writes=%d, want drops=2 writes=5", drops, underlying.writes)
	}
}

func TestRTPDropInterceptorDoesNotDropAudio(t *testing.T) {
	i := &rtpDropInterceptor{everyN: 2}
	underlying := &countWriter{}
	w := i.BindLocalStream(&interceptor.StreamInfo{MimeType: "audio/opus"}, underlying)
	for range 4 {
		_, _ = w.Write(&rtp.Header{}, []byte{1}, nil)
	}
	if underlying.writes != 4 {
		t.Fatalf("audio writes=%d, want 4", underlying.writes)
	}
}
