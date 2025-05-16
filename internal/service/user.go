package service

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"net/http"

	userrpc "liveclass/idl/kitex_gen/user"
)

func Register(c context.Context, ctx *app.RequestContext) {
	var user model.User

	err := ctx.BindJSON(&user)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, utils.H{
			"resp": model.Response{
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
			"resp": model.Response{
				Code: code.InternalError,
				Msg:  err.Error() + "rpc服务错误",
				Data: "nil",
			},
		})
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model.Response{
			Code: 0,
			Msg:  "ok",
			Data: rpcResp.Resp.Data,
		},
	})
}

func Login(c context.Context, ctx *app.RequestContext) (interface{}, error) {
	var user model.User

	err := ctx.BindJSON(&user)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model.Response{
				Code: code.BadRequest,
				Msg:  err.Error() + "参数错误",
				Data: "nil",
			},
		})
		return nil, nil
	}

	rpcResp, err := global.Clients.UserClient.Login(c, &userrpc.LoginReq{
		Username: user.Username,
		Password: user.Password,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.InternalError,
				Msg:  err.Error() + "rpc服务错误",
				Data: "nil",
			},
		})
		return nil, nil
	}

	return rpcResp.Resp.Data, nil
}

// 实际上是用的对外api
func GetUserInfo(c context.Context, ctx *app.RequestContext) {
	username := ctx.PostForm("username")

	rpcResp, err := global.Clients.UserClient.GetUserInfoByname(c, &userrpc.GetUserInfoByNameReq{
		Username: username,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.RPCError,
				Msg:  err.Error() + "userRpc error",
			},
		})
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model.Response{
			Code: 0,
			Msg:  "ok",
			Data: rpcResp.Resp.Data,
		},
	})
}

//对内id查询不对外
