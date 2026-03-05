// internal/rpc/webrtc_live/initialize/init_webrtc.go
package initialize

import (
	"sync"

	"github.com/pion/interceptor/pkg/report"

	"liveclass/internal/rpc/webrtc_live/global"
	"liveclass/internal/rpc/webrtc_live/model"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/webrtc/v4"
)

func InitWebRTCEngine() {
	// 媒体引擎
	me := &webrtc.MediaEngine{}
	if err := me.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}

	// 拦截器
	ir := &interceptor.Registry{}
	// 关键帧请求
	if pli, err := intervalpli.NewReceiverInterceptor(); err == nil && pli != nil {
		ir.Add(pli)
	}
	if senderFactory, err := report.NewSenderInterceptor(); err == nil && senderFactory != nil {
		ir.Add(senderFactory)
	}

	// 创建 API
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(me),
		webrtc.WithInterceptorRegistry(ir),
	)

	//PeerConnection 配置
	cfg := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	global.WebRTCEngine = &model.Engine{
		API:                 api,
		MediaEngine:         me,
		InterceptorRegistry: ir,
		SfuConfig:           cfg,
		BroadcastTracks:     sync.Map{},
	}
}
