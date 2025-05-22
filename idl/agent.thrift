namespace go agent

include "common.thrift"

struct ChatWithAgentReq{
    1:string userid,
    2:string message,
}

struct ChatWithAgentResp {
    1:common.resp resp,
}

service AgentService{
     ChatWithAgentResp ChatWithAgent(1: ChatWithAgentReq req),
}