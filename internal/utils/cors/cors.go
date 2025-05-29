package cors

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func CORS() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		origin := ctx.Request.Header.Get("Origin")
		if origin != "" {
			ctx.Response.Header.Add("Access-Control-Allow-Origin", "*")
			ctx.Response.Header.Add("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
			ctx.Response.Header.Add("Access-Control-Allow-Headers", "Authorization, Content-Type")
			ctx.Response.Header.Add("Access-Control-Allow-Credentials", "true")
		}
		if string(ctx.Request.Header.Method()) == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}
		ctx.Next(c)
	}
}
