package tool

import (
	"context"
	"errors"
	"fmt"

	"liveclass/idl/kitex_gen/common"
	user "liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	"liveclass/internal/rpc/agent/dependency"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type UserInfoRequest struct {
	UserID int64 `json:"user_id" jsonschema_description:"用户ID"`
}

type UserInfoResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Auth     string `json:"auth"`
	Status   int8   `json:"status"`
}

func NewUserInfoTool(cli userservice.Client) (tool.InvokableTool, error) {
	if cli == nil {
		return nil, errors.New("nil user service client")
	}

	call := func(ctx context.Context, req *UserInfoRequest) (*UserInfoResponse, error) {
		if req == nil {
			return nil, errors.New("empty request")
		}

		if req.UserID == 0 {
			return nil, errors.New("user_id is required")
		}

		info, err := fetchUserByID(ctx, cli, req.UserID)
		if err != nil {
			return nil, err
		}
		return &UserInfoResponse{
			UserID:   info.GetUserID(),
			Username: info.GetUserName(),
			Auth:     info.GetAuth(),
			Status:   info.GetStatus(),
		}, nil
	}

	return utils.InferTool("query_user_info", "查询指定用户的基础信息", call)
}

func fetchUserByID(ctx context.Context, cli userservice.Client, uid int64) (*common.User, error) {
	resp, err := dependency.Do(ctx, dependency.InternalRPC, "get_user_info", func(callCtx context.Context) (*user.GetUserInfoResp, error) {
		return cli.GetUserInfo(callCtx, &user.GetUserInfoReq{Userid: uid})
	})
	if err != nil {
		return nil, err
	}
	return parseUserResp(resp.GetResp())
}

func parseUserResp(resp *common.Resp) (*common.User, error) {
	if resp == nil {
		return nil, errors.New("user service: empty response")
	}
	if resp.GetCode() != 0 {
		return nil, fmt.Errorf("user service error: %s", resp.GetMsg())
	}
	if resp.GetData() == nil || resp.GetData().GetUserInfo() == nil {
		return nil, errors.New("user service: missing user info")
	}
	return resp.GetData().GetUserInfo(), nil
}
