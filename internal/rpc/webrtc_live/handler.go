package main

import (
	"context"
	"errors"
	"github.com/pion/webrtc/v4"
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/common"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/rpc/webrtc_live/global"
	my_webrtc "liveclass/internal/rpc/webrtc_live/webrtc"
	"log"
)

// WebrtcLiveImpl implements the last service interface defined in the IDL.
type WebrtcLiveImpl struct {
	DB *gorm.DB
}

// Broadcast implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) Broadcast(ctx context.Context, req *webrtc_live.BroadcastReq) (resp *webrtc_live.BroadcastResp, err error) {
	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(global.WebRTCEngine.MediaEngine),
		webrtc.WithInterceptorRegistry(global.WebRTCEngine.InterceptorRegistry),
	)
	pc, err := api.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	// 修改后的 AddTransceiver 逻辑
	_, err = pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	)
	if err != nil {
		return nil, err
	}

	videoTransceiver, err := pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	)
	if err != nil {
		return nil, err
	}

	// 确保 Codec 匹配
	videoTransceiver.SetCodecPreferences([]webrtc.RTPCodecParameters{
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}},
	})

	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		local, err := webrtc.NewTrackLocalStaticRTP(
			remote.Codec().RTPCodecCapability,
			"video"+req.LessonId,
			"stream"+req.LessonId,
		)
		if err != nil {
			panic(err)
		}
		global.WebRTCEngine.BroadcastTracks.Store(req.LessonId, local)
		log.Println("已存储到broadcast")
		buf := make([]byte, 1024)
		for {
			n, _, readErr := remote.Read(buf)
			if readErr != nil {
				return
			}
			if _, writeErr := local.Write(buf[:n]); writeErr != nil {
				continue
			}
		}
	})

	if err = pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	if err = pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-webrtc.GatheringCompletePromise(pc)

	b64ans, err := my_webrtc.EncodeSDP(pc.LocalDescription())
	if err != nil {
		return nil, err
	}
	log.Println(b64ans)
	return &webrtc_live.BroadcastResp{Resp: &common.Resp{Data: b64ans}}, nil
}

// View implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) View(ctx context.Context, req *webrtc_live.ViewReq) (resp *webrtc_live.ViewResp, err error) {
	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(global.WebRTCEngine.MediaEngine),
		webrtc.WithInterceptorRegistry(global.WebRTCEngine.InterceptorRegistry),
	)

	pc, err := api.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	v, ok := global.WebRTCEngine.BroadcastTracks.Load(req.LessonId)
	if !ok {
		return nil, errors.New("未开播:" + req.LessonId)
	}
	local := v.(*webrtc.TrackLocalStaticRTP)

	if _, err = pc.AddTrack(local); err != nil {
		return nil, err
	}

	if err = pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err = pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	<-webrtc.GatheringCompletePromise(pc)

	b64ans, err := my_webrtc.EncodeSDP(pc.LocalDescription())
	if err != nil {
		return nil, err
	}

	return &webrtc_live.ViewResp{Resp: &common.Resp{Data: b64ans}}, nil
}
