namespace go agent

include "common.thrift"

struct ChatWithAgentReq{
    1:i64 userid,
    2:string message,
    3:string request_id,
    4:string conv_id,
}

struct ChatWithAgentResp {
    1:common.resp resp,
}

struct ListAllUserConvReq {
    1: i64 userid,
}

struct ListAllUserConvResp{
    1:common.resp resp,
}

struct DelAllUserConvReq {
    1: i64 userid,
    2: string conv_id
}

struct DelAllUserConvResp {
    1:common.resp resp,
}

service AgentService{
    ChatWithAgentResp ChatWithAgent(1: ChatWithAgentReq req)(streaming.mode="server"),
    ListAllUserConvResp ListAllUserConv(1: ListAllUserConvReq req),
    DelAllUserConvResp DelAllUserConv(1: DelAllUserConvReq req),
}