namespace go chat

include "common.thrift"

struct CreateChatRoomReq {
    1: string userid,
    2: string lessonid,
}

struct CreateChatRoomResp {
    1:common.resp resp,
}

service ChatService {
    CreateChatRoomResp CreateChatRoomResp(1:CreateChatRoomReq req),
}