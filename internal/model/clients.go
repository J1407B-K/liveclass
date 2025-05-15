package model

import (
	live "liveclass/idl/kitex_gen/live/liveservice"
	user "liveclass/idl/kitex_gen/user/userservice"
)

type Clients struct {
	UserClient user.Client
	LiveClient live.Client
}
