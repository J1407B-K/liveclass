package jwt

import (
	"context"
	"errors"
	"fmt"
	userrpc "liveclass/idl/kitex_gen/user"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	jwtt "github.com/golang-jwt/jwt/v4"
	jwt "github.com/hertz-contrib/jwt"
)

var identityKey = "userid"
var jwtSecret = []byte("by_kq")

const ttl = 20 * time.Minute

func NewJWTMiddle() (*jwt.HertzJWTMiddleware, error) {
	authMiddlewire, err := jwt.New(&jwt.HertzJWTMiddleware{
		Realm:       "Hertz",
		Key:         jwtSecret,
		Timeout:     ttl,
		IdentityKey: identityKey,

		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(int64); ok {
				return jwt.MapClaims{
					identityKey: v,
				}
			}
			return jwt.MapClaims{}
		},

		IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
			claims := jwt.ExtractClaims(ctx, c)
			switch v := claims[identityKey].(type) {
			case float64:
				return int64(v)
			case int64:
				return v
			case string:
				id, _ := strconv.ParseInt(v, 10, 64)
				return id
			}
			return nil
		},

		Authenticator: func(c context.Context, ctx *app.RequestContext) (interface{}, error) {
			var user model2.User

			err := ctx.BindJSON(&user)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, utils.H{
					"resp": model2.Response{
						Code: code.BadRequest,
						Msg:  err.Error() + "参数错误",
						Data: "nil",
					},
				})
				return nil, errors.New("鉴权失败")
			}

			rpcResp, err := global.Clients.UserClient.Login(c, &userrpc.LoginReq{
				Username: user.Username,
				Password: user.Password,
			})
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, utils.H{
					"resp": model2.Response{
						Code: code.InternalError,
						Msg:  err.Error() + "rpc服务错误",
						Data: "nil",
					},
				})
				return nil, errors.New("鉴权失败")
			}
			if rpcResp == nil || rpcResp.Resp.Code != code.Success {
				return nil, errors.New("鉴权失败")
			}

			uid := rpcResp.Resp.Data.UserInfo.UserID
			ctx.Set("uid", uid)
			return uid, nil
		},

		LoginResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
			uid := c.GetInt64("uid")
			if uid == 0 {
				c.JSON(http.StatusInternalServerError, utils.H{
					"code":    http.StatusInternalServerError,
					"message": "missing uid",
				})
				return
			}

			refresh, err := GenerateRefreshToken(uid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, utils.H{
					"code":    http.StatusInternalServerError,
					"message": "refresh token error",
				})
				return
			}

			key := fmt.Sprintf("auth:refresh:%d", uid)
			err = global.DBManager.RDB.Set(ctx, key, refresh, 7*24*time.Hour).Err()
			if err != nil {
				c.JSON(http.StatusInternalServerError, utils.H{
					"code":    http.StatusInternalServerError,
					"message": "refresh token error",
				})
				return
			}

			c.JSON(http.StatusOK, utils.H{
				"code":          http.StatusOK,
				"message":       "success",
				"access_token":  token,
				"refresh_token": refresh,
				"expire":        expire.Unix(),
			})
		},

		Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
			c.JSON(http.StatusUnauthorized, utils.H{
				"code":    code,
				"message": message,
			})
		},
		ParseOptions: []jwtt.ParserOption{jwtt.WithValidMethods([]string{"HS256"})},
	})
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}
	return authMiddlewire, nil
}

func GenerateRefreshToken(userID int64) (string, error) {
	claims := jwtt.MapClaims{
		"userid": userID,
		"exp":    time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":   "refresh",
	}

	token := jwtt.NewWithClaims(jwtt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseRefreshToken(tokenString string) (int64, error) {
	token, err := jwtt.Parse(tokenString, func(t *jwtt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return 0, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwtt.MapClaims)
	if !ok {
		return 0, errors.New("invalid token")
	}
	if tp, _ := claims["type"].(string); tp != "refresh" {
		return 0, errors.New("invalid refresh token")
	}
	return int64(claims["userid"].(float64)), nil
}

func GenerateAccessToken(uid int64) (string, time.Time, error) {
	exp := time.Now().Add(ttl)
	claims := jwtt.MapClaims{
		"userid": uid,
		"exp":    exp.Unix(),
		"type":   "access",
	}
	t := jwtt.NewWithClaims(jwtt.SigningMethodHS256, claims)
	s, err := t.SignedString(jwtSecret)
	return s, exp, err
}

func ParseAccessToken(tokenStr string) (int64, error) {
	token, err := jwtt.Parse(tokenStr, func(token *jwtt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}

	claims := token.Claims.(jwtt.MapClaims)

	uid := int64(claims["userid"].(float64))
	return uid, nil
}
