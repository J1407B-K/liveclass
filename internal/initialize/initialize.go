package initialize

import (
	"github.com/cloudwego/kitex/client"
	"liveclass/idl/kitex_gen/live/liveservice"
	"liveclass/idl/kitex_gen/user/userservice"
	"liveclass/internal/global"
)

func InitNewClient() error {
	uc, err := userservice.NewClient("userservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.UserClient = uc

	lc, err := liveservice.NewClient("liveservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.LiveClient = lc

	return nil
}
