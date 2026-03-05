package service

import (
	"context"
	"fmt"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"liveclass/internal/api/utils/jwt"
	"net/http"
	"time"

	userrpc "liveclass/idl/kitex_gen/user"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func Register(c context.Context, ctx *app.RequestContext) {
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
		return
	}

	rpcResp, err := global.Clients.UserClient.Register(c, &userrpc.RegisterReq{
		Username: user.Username,
		Password: user.Password,
		Auth:     user.Auth,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.InternalError,
				Msg:  err.Error() + "rpc服务错误",
				Data: "nil",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model2.Response{
			Code: 0,
			Msg:  "ok",
			Data: rpcResp.Resp.Data,
		},
	})
}

func RefreshToken(c context.Context, ctx *app.RequestContext) {
	refresh := ctx.PostForm("refresh_token")

	uid, err := jwt.ParseRefreshToken(refresh)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.BadRequest,
				Msg:  err.Error(),
			},
		})
		return
	}

	key := fmt.Sprintf("auth:refresh:%d", uid)
	val, err := global.DBManager.RDB.Get(c, key).Result()
	if err != nil || val != refresh {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{Code: code.AuthError, Msg: "refresh token invalid"},
		})
		return
	}

	access, accessExp, err := jwt.GenerateAccessToken(uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{Code: code.InternalError, Msg: "access token error"},
		})
		return
	}

	newRefreshToken, err := jwt.GenerateRefreshToken(uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{Code: code.InternalError, Msg: "refresh token error"},
		})
		return
	}

	_ = global.DBManager.RDB.Set(c, key, newRefreshToken, 7*24*time.Hour).Err()

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model2.Response{
			Code: code.Success,
			Msg:  "ok",
			Data: utils.H{
				"access_token":  access,
				"access_expire": accessExp.Unix(),
				"refresh_token": newRefreshToken,
			},
		},
	})
}

func GetUserInfo(c context.Context, ctx *app.RequestContext) {
	username := ctx.PostForm("username")

	rpcResp, err := global.Clients.UserClient.GetUserInfoByname(c, &userrpc.GetUserInfoByNameReq{
		Username: username,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.RPCError,
				Msg:  err.Error() + "userRpc error",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model2.Response{
			Code: 0,
			Msg:  "ok",
			Data: rpcResp.Resp.Data,
		},
	})
}
