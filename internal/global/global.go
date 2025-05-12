package global

import (
	"github.com/cloudwego/kitex/pkg/discovery"
	"liveclass/internal/model"
)

var (
	Clients  = model.Clients{}
	Resolver *discovery.Resolver
)
