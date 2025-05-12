package model

import (
	user "liveclass/idl/kitex_gen/user/userservice"
)

type Clients struct {
	UserClient user.Client
}
