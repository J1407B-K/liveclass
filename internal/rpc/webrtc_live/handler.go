// internal/rpc/webrtc_live/service/webrtc_live_impl.go
package main

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"log"

	"github.com/pion/webrtc/v4"
	"liveclass/idl/kitex_gen/common"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/rpc/webrtc_live/global"
	my_webrtc "liveclass/internal/rpc/webrtc_live/webrtc"
)

// WebrtcLiveImpl implements the Kitex webrtc_live service
type WebrtcLiveImpl struct {
	DB *gorm.DB
}

func (s *WebrtcLiveImpl) Broadcast(ctx context.Context, req *webrtc_live.BroadcastReq) (*webrtc_live.BroadcastResp, error) {
	// decode browser offer
	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	// new PeerConnection (default host+srflx candidates)
	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	// recvonly audio+video
	pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	vt, _ := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	vt.SetCodecPreferences([]webrtc.RTPCodecParameters{
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}},
	})

	// ICE 日志
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][Broadcast] ICE candidate:", c.ToJSON())
		}
	})
	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		log.Println("[Server][Broadcast] ICE state:", s.String())
	})

	// OnTrack: 收到浏览器的 track，创建本地转发 track 并缓存
	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Println("[Server][Broadcast] OnTrack kind=", remote.Kind())
		local, err := webrtc.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability,
			remote.Kind().String()+"-"+req.LessonId,
			"stream-"+req.LessonId,
		)
		if err != nil {
			log.Println("track create error:", err)
			return
		}
		raw, _ := global.WebRTCEngine.BroadcastTracks.LoadOrStore(req.LessonId, []*webrtc.TrackLocalStaticRTP{})
		arr := raw.([]*webrtc.TrackLocalStaticRTP)
		arr = append(arr, local)
		global.WebRTCEngine.BroadcastTracks.Store(req.LessonId, arr)

		// 转发 RTP 包
		buf := make([]byte, 1500)
		for {
			n, _, readErr := remote.Read(buf)
			if readErr != nil {
				return
			}
			local.Write(buf[:n]) // 忽略写错误
		}
	})

	// SDP 交换
	if err := pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	<-webrtc.GatheringCompletePromise(pc)

	b64ans, _ := my_webrtc.EncodeSDP(pc.LocalDescription())
	return &webrtc_live.BroadcastResp{Resp: &common.Resp{Data: b64ans}}, nil
}

func (s *WebrtcLiveImpl) View(ctx context.Context, req *webrtc_live.ViewReq) (*webrtc_live.ViewResp, error) {
	// decode browser offer
	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][View] ICE candidate:", c.ToJSON())
		}
	})
	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		log.Println("[Server][View] ICE state:", s.String())
	})

	if err := pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	raw, ok := global.WebRTCEngine.BroadcastTracks.Load(req.LessonId)
	if !ok {
		return nil, errors.New("未开播: " + req.LessonId)
	}
	for _, t := range raw.([]*webrtc.TrackLocalStaticRTP) {
		pc.AddTrack(t)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	<-webrtc.GatheringCompletePromise(pc)

	b64ans, _ := my_webrtc.EncodeSDP(pc.LocalDescription())
	return &webrtc_live.ViewResp{Resp: &common.Resp{Data: b64ans}}, nil
}
