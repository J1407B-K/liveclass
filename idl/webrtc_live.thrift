namespace go webrtc_live

include "common.thrift"

struct BroadcastReq{
    1:string userid,
    2:string lesson_id,
    3:string b64offer,
}

struct BroadcastResp{
    1:common.resp resp,
}

struct ViewReq{
    1: string userid,
    2: string lesson_id,
    3: string b64offer,
}

struct ViewResp{
    1:common.resp resp,
}

struct CreateLessonReq {
    1: string userid,
    2: string lesson_name,
    3: string description,
}

struct CreateLessonResp {
    1:common.resp resp,
}

struct DelLessonReq {
    1: string userid,
    2: string lessonid,
}

struct DelLessonResp {
    1: common.resp resp,
}

struct ChangeUserInLiveReq {
    1: string lessonid,
    2: string userid,
    3: string options,
}

struct ChangeUserInLiveResp {
    1: common.resp resp,
}

struct ChangeUserToLessonReq{
    1: string userid,
    2: string lessonid,
    3: string options,
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

struct SelectLessonInfoReq{
    1: string lessonid,
}

struct SelectLessonInfoResp{
    1:common.resp resp,
}

struct GetLessonInfoReq{
    1: string lesson_name,
    2: string teacher,
}

struct GetLessonInfoResp{
    1: common.resp resp,
}

struct IsStudentInLessonReq {
    1: string studentid,
    2: string lessonid,
}

struct IsStudentInLessonResp {
    1:common.resp resp,
}

struct CreateSignInReq {
    1: string userid,
    2: string lessonid,
    3: i64  duration,
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

struct SelectSignInReq{
    1:string userid,
    2:string lessonid,
}

struct SelectSignInResp{
    1:common.resp resp,
}

struct DelSignInReq{
    1:string userid,
    2:string lessonid,
}

struct DelSignInResp{
    1:common.resp resp,
}

struct RollCallInRandomReq{
    1:string userid,
    2:string lesson_id
}

struct RollCallInRandomResp{
    1:common.resp resp,
}

struct RecordLessonReq {
    1: string userid,
    2: string lessonid,
    3: binary data,
}

struct RecordLessonResp {
    1:common.resp resp,
}

struct SaveWhiteBoardJsonReq {
    1: string userid,
    2: string lessonid,
    3: string file,
}

struct SaveWhiteBoardJsonResp {
    1: common.resp resp,
}

struct GetWhiteBoardJsonReq {
    1: string userid,
    2: string lessonid,
}

struct GetWhiteBoardJsonResp{
    1:common.resp resp,
}

struct PublishMicReq {
    1: string userid,
    2: string lessonid,
    3: string b64offer,
}

struct PublishMicResp {
    1: common.resp resp,
}

struct RaiseHandReq {
    1: string userid,
    2: string lessonid,
}

struct RaiseHandResp {
    1:common.resp resp,
}

struct GetRaiseHandReq{
    1: string userid,
    2: string lessonid,
}

struct GetRaiseHandResp {
    1:common.resp resp,
}

struct ApproveHandReq{
    1:string userid,
    2:string lessonid,
    3:string stuid,
}

struct ApproveHandResp{
    1:common.resp resp,
}

struct ViewMicReq{
    1:string userid,
    2:string lessonid,
    3:string b64offer,
}

struct ViewMicResp{
    1:common.resp resp,
}

struct ListAllLessonRecordReq{
    1:string userid,
    2:string lessonid,
}

struct ListAllLessonRecordResp{
    1:common.resp resp,
}

struct GetLessonRecordReq{
    1:string userid,
    2:string lessonid,
    3:string key,
}

struct GetLessonRecordResp{
    1: binary data,
}

service webrtc_live {
    BroadcastResp Broadcast(1: BroadcastReq req),
    ViewResp      View(1: ViewReq req),
    CreateLessonResp  CreateLesson(1: CreateLessonReq req),
    DelLessonResp     DelLesson(1: DelLessonReq req),
    ChangeUserInLiveResp ChangeUserInLive(1: ChangeUserInLiveReq req),
    ChangeUserToLessonResp ChangeUserToLesson(1: ChangeUserToLessonReq req),
    GetLessonInfoByIdResp  GetLessonInfoById(1: GetLessonInfoByIdReq req),
    SelectLessonInfoResp   SelectLessonInfo(1: SelectLessonInfoReq req),
    GetLessonInfoResp      GetLessonInfo(1: GetLessonInfoReq req),
    IsStudentInLessonResp  IsStudentInLesson(1: IsStudentInLessonReq req),
    CreateSignInResp    CreateSignIn(1:CreateSignInReq req),
    SignInResp          SignIn(1:SignInReq req),
    SelectSignInResp    SelectSignIn(1: SelectSignInReq req),
    DelSignInResp       DelSign(1:DelSignInReq req),
    RollCallInRandomResp    RollCallInRandom(1:RollCallInRandomReq req),
    RecordLessonResp      RecordLesson(1:RecordLessonReq req),
    SaveWhiteBoardJsonResp SaveWhiteBoardJson(1: SaveWhiteBoardJsonReq req),
    GetWhiteBoardJsonResp  GetWhiteBoardJson(1:GetWhiteBoardJsonReq req),
    PublishMicResp         PublishMic(1: PublishMicReq req),
    RaiseHandResp          RaiseHand(1: RaiseHandReq req),
    GetRaiseHandResp       GetRaiseHand(1: GetRaiseHandReq req),
    ApproveHandResp        ApproveHand(1: ApproveHandReq req),
    ViewMicResp            ViewMic(1: ViewMicReq req),
    ListAllLessonRecordResp ListAllLessonRecord(1: ListAllLessonRecordReq req),
    GetLessonRecordResp      GetLessonRecord(1:GetLessonRecordReq req),
}
