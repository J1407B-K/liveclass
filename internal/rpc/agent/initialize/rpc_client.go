package initialize

import (
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	userservice "liveclass/idl/kitex_gen/user/userservice"
	webrtclive "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/agent/global"
)

func InitUserClient() (userservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		return nil, err
	}
	return userservice.NewClient("userservice",
		client.WithResolver(r),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithRPCTimeout(global.Config.Resilience.InternalRPC.Timeout),
	)
}

func InitLessonClient() (webrtclive.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		return nil, err
	}
	return webrtclive.NewClient("webrtc_liveservice",
		client.WithResolver(r),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithRPCTimeout(global.Config.Resilience.InternalRPC.Timeout),
	)
}
