// internal/rpc/webrtc_live/initialize/init_webrtc.go
package initialize

import (
	"net"
	"sync"

	"liveclass/internal/rpc/webrtc_live/global"
	"liveclass/internal/rpc/webrtc_live/model"

	"github.com/pion/interceptor"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

func InitWebRTCEngine(onInjectedDrop func()) {
	me := &webrtc.MediaEngine{}
	if err := me.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}

	ir := &interceptor.Registry{}
	if global.Config.RTPDropEveryN > 0 {
		ir.Add(&rtpDropFactory{everyN: uint64(global.Config.RTPDropEveryN), onDrop: onInjectedDrop})
	}
	if global.Config.NACKEnabled {
		if err := webrtc.ConfigureNack(me, ir); err != nil {
			panic(err)
		}
	}
	if err := webrtc.ConfigureRTCPReports(ir); err != nil {
		panic(err)
	}
	if err := webrtc.ConfigureSimulcastExtensionHeaders(me); err != nil {
		panic(err)
	}
	if err := webrtc.ConfigureTWCCSender(me, ir); err != nil {
		panic(err)
	}

	settingEngine := webrtc.SettingEngine{}
	if global.Config.ICEUDPAddr != "" {
		udpConn, err := net.ListenPacket("udp", global.Config.ICEUDPAddr)
		if err != nil {
			panic(err)
		}
		logger := logging.NewDefaultLoggerFactory().NewLogger("ice-udp-mux")
		settingEngine.SetICEUDPMux(webrtc.NewICEUDPMux(logger, udpConn))
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(me),
		webrtc.WithInterceptorRegistry(ir),
		webrtc.WithSettingEngine(settingEngine),
	)

	// STUN
	cfg := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	global.WebRTCEngine = &model.Engine{
		API:                 api,
		MediaEngine:         me,
		InterceptorRegistry: ir,
		SettingEngine:       settingEngine,
		SfuConfig:           cfg,
		BroadcastRooms:      sync.Map{},
	}
}
