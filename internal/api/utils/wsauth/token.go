package wsauth

import (
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// Token avoids putting credentials in URLs in production. Authorization and
// HttpOnly cookie transports are preferred; query transport is an explicit
// development compatibility switch.
func Token(ctx *app.RequestContext, allowQuery bool) string {
	authorization := strings.TrimSpace(string(ctx.Request.Header.Peek("Authorization")))
	if len(authorization) > 7 && strings.EqualFold(authorization[:7], "Bearer ") {
		if token := strings.TrimSpace(authorization[7:]); token != "" {
			return token
		}
	}
	if token := strings.TrimSpace(string(ctx.Cookie("access_token"))); token != "" {
		return token
	}
	if allowQuery {
		return strings.TrimSpace(ctx.Query("token"))
	}
	return ""
}
