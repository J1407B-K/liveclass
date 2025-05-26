namespace go webrtc_live

include "common.thrift"

struct BroadcastReq{
    1:string lesson_id,
    2:string b64offer,
}

struct BroadcastResp{
    1:common.resp resp,
}

struct ViewReq{
    1: string lesson_id,
    2: string b64offer,
}

struct ViewResp{
    1:common.resp resp,
}

struct CreateLiveReq {
    1: string userid,
    2: string lesson_name,
}

struct CreateLiveResp {
    1:common.resp resp,
}

struct DelLiveReq {
    1: string userid,
    2: string lessonid,
}

struct DelLiveResp {
    1: common.resp resp,
}

struct ChangeUserInLiveReq {
    1: string livename,
    2: string userid,
    3: string options,
}

struct ChangeUserInLiveResp {
    1: common.resp resp,
}

struct ChangeUserToLessonReq{
    1: string userid,
    2: string lessonid,
}

struct ChangeUserToLessonResp {
    1:common.resp resp,
}

struct GetLessonInfoByIdReq {
    1: string lessonid
}

struct GetLessonInfoByIdResp{
    1:common.resp resp,
}

service webrtc_live {
    BroadcastResp Broadcast(1: BroadcastReq req),
    ViewResp      View(1: ViewReq req),
    CreateLiveResp  CreateLive(1: CreateLiveReq req),
    DelLiveResp     DelLive(1: DelLiveReq req),
    ChangeUserInLiveResp ChangeUserInLive(1: ChangeUserInLiveReq req),
    ChangeUserToLessonResp ChangeUserToLesson(1: ChangeUserToLessonReq req),
    GetLessonInfoByIdResp  GetLessonInfoById(1: GetLessonInfoByIdReq req),
}
