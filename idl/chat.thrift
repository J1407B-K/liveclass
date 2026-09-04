namespace go chat

include "common.thrift"

struct LiveChatReq {
    1: i64 userid,
    2: i64 lessonid,
    3: string message,
    4: optional string client_message_id,
}

struct LiveChatResp {
    1:common.resp resp,
    2:optional string message_id,
    3:optional string delivery_status,
}

struct GetHistoryReq {
    1: i64 lesson_id,
    2: i64 userid,
    3:optional string cursor,
    4:optional i32 limit,
    5:optional string after_message_id,
}

struct GetHistoryResp {
    1: common.resp resp,
    2:optional string next_cursor,
    3:optional bool has_more,
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
