// internal/rpc/webrtc_live/service/webrtc_live_impl.go
package main

import (
	"context"
	"errors"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/common"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/rpc/webrtc_live/global"
	my_webrtc "liveclass/internal/rpc/webrtc_live/webrtc"
	"log"
	"time"
)

// WebrtcLiveImpl implements the Kitex webrtc_live service
type WebrtcLiveImpl struct {
	DB *gorm.DB
}

func (s *WebrtcLiveImpl) Broadcast(ctx context.Context, req *webrtc_live.BroadcastReq) (*webrtc_live.BroadcastResp, error) {
	//先解码
	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	//全局配置创建pc
	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	//需要的类型
	pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	vt, _ := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	vt.SetCodecPreferences([]webrtc.RTPCodecParameters{
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}},
	})

	// 日志
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][Broadcast] ICE candidate:", c.ToJSON())
		}
	})
	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		log.Println("[Server][Broadcast] ICE state:", s.String())
	})

	//收到流时的逻辑
	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Println("[Server][Broadcast] OnTrack kind=", remote.Kind())
		local, err := webrtc.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability, //编码器
			remote.Kind().String()+"-"+req.LessonId, //trackID
			"stream-"+req.LessonId,                  //媒体流名称，归类用
		)
		if err != nil {
			log.Println("track create error:", err)
			return
		}

		//存储
		raw, _ := global.WebRTCEngine.BroadcastTracks.LoadOrStore(req.LessonId, []*webrtc.TrackLocalStaticRTP{})
		arr := raw.([]*webrtc.TrackLocalStaticRTP)
		arr = append(arr, local)
		global.WebRTCEngine.BroadcastTracks.Store(req.LessonId, arr)

		//不断请求关键帧
		go func(ssrc uint32) {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := pc.WriteRTCP([]rtcp.Packet{
					&rtcp.PictureLossIndication{MediaSSRC: ssrc},
				}); err != nil {
					// PC 关闭或出错就停止
					return
				}
			}
		}(uint32(remote.SSRC()))

		for {
			pkt, _, readErr := remote.ReadRTP()
			if readErr != nil {
				return
			}

			if writeErr := local.WriteRTP(pkt); writeErr != nil {
				return
			}
		}
	})

	if err := pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	//连接断开或出错时，从缓存中删除
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("[Server][Broadcast] PeerConnection state:", state.String())
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed ||
			state == webrtc.PeerConnectionStateDisconnected {
			log.Println("[Server][Broadcast] Cleaning up maps for lesson", req.LessonId)
			global.WebRTCEngine.BroadcastTracks.Delete(req.LessonId)
		}
	})

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	<-webrtc.GatheringCompletePromise(pc)

	//编码
	b64ans, _ := my_webrtc.EncodeSDP(pc.LocalDescription())
	return &webrtc_live.BroadcastResp{Resp: &common.Resp{Data: b64ans}}, nil
}

func (s *WebrtcLiveImpl) View(ctx context.Context, req *webrtc_live.ViewReq) (*webrtc_live.ViewResp, error) {
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
	//将轨道加入view pc
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
