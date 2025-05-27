package initialize

import (
	"liveclass/internal/rpc/webrtc_live/global"
	"net/http"
	"net/url"

	"github.com/tencentyun/cos-go-sdk-v5"
)

func SetUpCos() *cos.Client {
	u, _ := url.Parse("https://" + global.Config.CosConfig.BucketnameAppid + ".cos." + global.Config.CosConfig.CosRegion + ".myqcloud.com")
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  global.Config.CosConfig.SecretId,
			SecretKey: global.Config.CosConfig.SecretKey,
		},
	})

	return client
}
