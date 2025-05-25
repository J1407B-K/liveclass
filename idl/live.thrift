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
    3: string options,
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

struct ChangeUserToLessonReq {
    1: string userid,
    2: string lessonname,
    3: string teacher,
    4: string option,
    5: string studentid,
}

struct ChangeUserToLessonResp {
    1: common.resp resp,
}

struct IsStudentInLessonReq {
    1: string studentid,
    2: string lessonid,
}

struct IsStudentInLessonResp {
    1:common.resp resp,
}

struct RecordLessonReq{
    1: string userid,
    2: string streamURL,
    3: string lesson_id,
    4: i32    duration,
}

struct RecordLessonResp {
    1:common.resp resp,
}

struct CreateSignInReq {
    1: string userid,
    2: string lessonid,
}

struct CreateSignInResp {
    1:common.resp resp,
}

struct SignInReq{
    1:string userid,
    2:string lessonid,
}

struct SignInResp {
    1:common.resp resp,
}

service LiveService {
    CreateLiveResp CreateLive(1: CreateLiveReq req),
    CloseLiveResp  CloseLive(1:CloseLiveReq req),
   ChangeUserInLiveResp ChangeUserInLive(1: ChangeUserInLiveReq req),
   SelectLessonInfoResp SelectLessonInfo(1:SelectLessonInfoReq req),
   GetLessonInfoResp    GetLessonInfo(1:GetLessonInfoReq req),
   GetLessonInfoByIdResp GetLessonInfoById(1:GetLessonInfoByIdReq req),
   ChangeUserToLessonResp   ChangeUserToLesson(1:ChangeUserToLessonReq req),
   IsStudentInLessonResp IsStudentInLesson(1:IsStudentInLessonReq req),
   RecordLessonResp     RecordLesson(1:RecordLessonReq req),
    CreateSignInResp    CreateSignIn(1:CreateSignInReq req),
    SignInResp          SignIn(1:SignInReq req),
}
