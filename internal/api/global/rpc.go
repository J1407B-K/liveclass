package global

import (
	"liveclass/internal/api/model"

	"github.com/cloudwego/kitex/pkg/discovery"
)

var (
	Clients  = model.Clients{}
	Resolver *discovery.Resolver
)
