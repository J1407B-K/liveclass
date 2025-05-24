namespace go chat

include "common.thrift"

struct LiveChatReq {
    1: string userid,
    2: string lessonid,
    3: string message,
}

struct LiveChatResp {
    1:common.resp resp,
}

struct GetHistoryReq {
    1: string lesson_id,
}

struct GetHistoryResp {
    1: common.resp resp,
}

struct DelHistoryReq{
    1: string lesson_id,
}

struct DelHistoryResp {
    1: common.resp resp,
}

service ChatService {
    LiveChatResp LiveChat(1: LiveChatReq req),
    GetHistoryResp GetHistory(1: GetHistoryReq req),
    DelHistoryResp DelHistory(1: DelHistoryReq req),
}