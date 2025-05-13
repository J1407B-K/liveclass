namespace go user

include "common.thrift"

struct RegisterReq{
    1: string username,
    2: string password,
    3: string auth,
    4: string lessons,
}

struct RegisterResp{
    1: common.resp resp
}

struct LoginReq{
    1: string username,
    2: string password,
    3: string auth
}

struct LoginResp{
    1: common.resp resp
}

struct GetUserInfoReq{
    1: string username,
}

struct GetUserInfoResp{
    1:  common.resp resp,
}


service UserService{
    RegisterResp Register(1: RegisterReq req),
    LoginResp Login(1: LoginReq req),
    GetUserInfoResp  GetUserInfo(1: GetUserInfoReq req)
}