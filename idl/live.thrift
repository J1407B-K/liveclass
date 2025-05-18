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

struct ChangeUserInLiveReq {
    1: string livename,
    2: string userid,
    3:string options,
}

struct ChangeUserInLiveResp {
    1: common.resp resp,
}


struct SelectLessonInfoReq{
    1: string lessonname,
    2: string teacher,
}

struct SelectLessonInfoResp {
    1: common.resp resp,
}

struct GetLessonInfoReq {
    1: string lessonname,
    2: string teacher,
}

struct GetLessonInfoResp {
    1: common.resp resp,
}

struct GetLessonInfoByIdReq{
    1: string lessonid
}

struct GetLessonInfoByIdResp{
    1: common.resp resp,
}

service LiveService {
    CreateLiveResp CreateLive(1: CreateLiveReq req),
    CloseLiveResp  CloseLive(1:CloseLiveReq req),
   ChangeUserInLiveResp ChangeUserInLive(1: ChangeUserInLiveReq req),
   SelectLessonInfoResp SelectLessonInfo(1:SelectLessonInfoReq req),
   GetLessonInfoResp    GetLessonInfo(1:GetLessonInfoReq req),
   GetLessonInfoByIdResp GetLessonInfoById(1:GetLessonInfoByIdReq req)
}
