namespace go chat

include "common.thrift"

struct LiveChatReq {
    1: i64 userid,
    2: i64 lessonid,
    3: string message,
}

struct LiveChatResp {
    1:common.resp resp,
    2:optional string message_id,
}

struct GetHistoryReq {
    1: i64 lesson_id,
    2: i64 userid,
    3:optional string cursor,
    4:optional i32 limit,
}

struct GetHistoryResp {
    1: common.resp resp,
    2:optional string next_cursor,
}

struct DelHistoryReq{
    1: i64 lesson_id,
    2: i64 userid,
}

struct DelHistoryResp {
    1: common.resp resp,
}

service ChatService {
    LiveChatResp LiveChat(1: LiveChatReq req),
    GetHistoryResp GetHistory(1: GetHistoryReq req),
    DelHistoryResp DelHistory(1: DelHistoryReq req),
}
