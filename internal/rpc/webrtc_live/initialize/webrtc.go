// internal/rpc/webrtc_live/initialize/init_webrtc.go
package initialize

import (
	"github.com/pion/interceptor/pkg/report"
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/webrtc/v4"
	"liveclass/internal/rpc/webrtc_live/global"
	"liveclass/internal/rpc/webrtc_live/model"
)

func InitWebRTCEngine() {
	// 1. 媒体引擎 + 默认编解码器
	me := &webrtc.MediaEngine{}
	if err := me.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}

	// 2. 拦截器 + 默认拦截器
	ir := &interceptor.Registry{}
	//    加一个 PLI interceptor（每 3 秒发一帧关键帧请求）
	if pli, err := intervalpli.NewReceiverInterceptor(); err == nil && pli != nil {
		ir.Add(pli)
	}
	if senderFactory, err := report.NewSenderInterceptor(); err == nil && senderFactory != nil {
		ir.Add(senderFactory)
	}

	// 3. 用默认 SettingEngine / MediaEngine / InterceptorRegistry 创建 API
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(me),
		webrtc.WithInterceptorRegistry(ir),
	)

	// 4. PeerConnection 配置：用官方 STUN（局域网也能用 host cand）
	cfg := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	// 5. 保存到全局
	global.WebRTCEngine = &model.Engine{
		API:                 api,
		MediaEngine:         me,
		InterceptorRegistry: ir,
		SfuConfig:           cfg,
		BroadcastTracks:     sync.Map{},
	}
}
