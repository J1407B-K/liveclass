package main

import (
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"log"
)

func main() {
	svr := webrtc_live.NewServer(new(WebrtcLiveImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
