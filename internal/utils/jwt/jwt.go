package jwt

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/jwt"
	"liveclass/internal/model"
	"liveclass/internal/service"
	"log"
	"net/http"
	"time"
)

var identityKey = "userid"

func NewMiddle() (*jwt.HertzJWTMiddleware, error) {
	authMiddlewire, err := jwt.New(&jwt.HertzJWTMiddleware{
		Realm:       "Hertz",
		Key:         []byte("by_kq"),
		Timeout:     24 * time.Hour,
		IdentityKey: identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(string); ok {
				return jwt.MapClaims{
					identityKey: v,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
			claims := jwt.ExtractClaims(ctx, c)
			if userid, ok := claims[identityKey].(string); ok {
				return &model.User{UserId: userid}
			}
			return nil // 避免 nil 指针
		},
		Authenticator: service.Login,
		Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
			c.JSON(http.StatusUnauthorized, utils.H{
				"code":    code,
				"message": message,
			})
		},
	})
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}
	return authMiddlewire, nil
}
