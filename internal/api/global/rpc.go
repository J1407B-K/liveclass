package global

import (
	"liveclass/internal/api/model"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/discovery"
)

var (
	Clients  = model.Clients{}
	Resolver *discovery.Resolver
	Node     *snowflake.Node
)
