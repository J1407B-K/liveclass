package main

import (
	"context"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
)

// WebrtcLiveImpl implements the last service interface defined in the IDL.
type WebrtcLiveImpl struct{}

// Broadcast implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) Broadcast(ctx context.Context, req *webrtc_live.BroadcastReq) (resp *webrtc_live.BroadcastResp, err error) {
	// TODO: Your code here...
	return
}

// View implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) View(ctx context.Context, req *webrtc_live.ViewReq) (resp *webrtc_live.ViewResp, err error) {
	// TODO: Your code here...
	return
}
