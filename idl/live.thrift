namespace go live

include "common.thrift"

struct CreateLiveReq {
    1: string livename,
    2: string userid,
    3: string description,
}

struct CreateLiveResp {
    1: common.resp resp,
}

struct CloseLiveReq {
    1: string livename,
    2: string userid,
}

struct CloseLiveResp {
    1: common.resp resp,
}

struct AddUserInLiveReq {
    1: string livename,
    2: string userid,
}

struct AddUserInLiveResp {
    1: common.resp resp,
}

struct DelUserInLiveReq {
    1: string livename,
    2: string userid,
}

struct DelUserInLiveResp {
    1: common.resp resp,
}

service LiveService {
    CreateLiveResp CreateLive(1: CreateLiveReq req),
    CloseLiveResp  CloseLive(1:CloseLiveReq req),
    AddUserInLiveResp AddUserInLive(1:AddUserInLiveReq req),
    DelUserInLiveResp DelUserInlive(1:DelUserInLiveReq req),
}
