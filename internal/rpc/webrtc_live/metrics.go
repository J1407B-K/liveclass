package main

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	webrtcActivePeers        = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "webrtc_active_peer_connections", Help: "Current WebRTC peer connections by role."}, []string{"role"})
	webrtcPeerStates         = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "webrtc_peer_connection_state_total", Help: "Peer connection state transitions by role and state."}, []string{"role", "state"})
	webrtcRTPPacketsIn       = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "webrtc_rtp_packets_in_total", Help: "RTP packets received from publishers by media kind."}, []string{"kind"})
	webrtcRTPPacketsOut      = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "webrtc_rtp_packets_out_total", Help: "RTP packets accepted by the local fanout track by media kind."}, []string{"kind"})
	webrtcRTPWriteErrors     = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "webrtc_rtp_write_errors_total", Help: "RTP fanout write errors by media kind."}, []string{"kind"})
	webrtcNACKReceived       = prometheus.NewCounter(prometheus.CounterOpts{Name: "webrtc_nack_received_total", Help: "RTP sequence numbers requested by viewer NACK feedback."})
	webrtcPLIReceived        = prometheus.NewCounter(prometheus.CounterOpts{Name: "webrtc_pli_received_total", Help: "PLI or FIR keyframe requests received from viewers."})
	webrtcPLIForwarded       = prometheus.NewCounter(prometheus.CounterOpts{Name: "webrtc_pli_forwarded_total", Help: "Aggregated keyframe requests forwarded to publishers."})
	webrtcPLISuppressed      = prometheus.NewCounter(prometheus.CounterOpts{Name: "webrtc_pli_suppressed_total", Help: "Viewer keyframe requests suppressed by room-level aggregation."})
	webrtcRTPInjectedDrops   = prometheus.NewCounter(prometheus.CounterOpts{Name: "webrtc_rtp_injected_drops_total", Help: "RTP packets dropped by the deterministic benchmark fault injector."})
	webrtcTrackReadyTimeouts = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "webrtc_track_ready_timeout_total", Help: "Viewer setup timeouts waiting for publisher tracks."}, []string{"kind"})
)

func registerWebRTCMetrics(reg prometheus.Registerer) {
	reg.MustRegister(webrtcActivePeers, webrtcPeerStates, webrtcRTPPacketsIn, webrtcRTPPacketsOut, webrtcRTPWriteErrors, webrtcNACKReceived, webrtcPLIReceived, webrtcPLIForwarded, webrtcPLISuppressed, webrtcRTPInjectedDrops, webrtcTrackReadyTimeouts)
}

type peerLifecycle struct {
	role string
	once sync.Once
}

func newPeerLifecycle(role string) *peerLifecycle {
	webrtcActivePeers.WithLabelValues(role).Inc()
	return &peerLifecycle{role: role}
}

func (p *peerLifecycle) observe(state string, terminal bool) {
	webrtcPeerStates.WithLabelValues(p.role, state).Inc()
	if terminal {
		p.close()
	}
}

func (p *peerLifecycle) close() {
	p.once.Do(func() { webrtcActivePeers.WithLabelValues(p.role).Dec() })
}
