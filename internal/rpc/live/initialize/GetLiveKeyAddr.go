package initialize

import "liveclass/internal/rpc/live/global"

func InitKeyAddr() string {
	return global.Config.GetLiveKeyAddr
}
