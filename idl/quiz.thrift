namespace go quiz

include "common.thrift"

struct CreateQuestionReq {
    1: string LessonId,
    2: string Userid,
    3: string Content,
    4: i32 OptionsNum,
    5: list<string> Options,
    6: string Answer,
    7: i32 duration,
}

struct CreateQuestionResp {
    1: common.resp resp,
}

struct TorFAnswerReq {
    1: string Question_id,
    2: string Userid,
    3: string UserAnswer,
}

struct TorFAnswerResp {
    1: common.resp resp,
}

struct DelQuestionReq {
    1:string Userid,
    2:string Question_id,
}

struct DelQuestionResp {
    1: common.resp resp,
}

struct GetAllLessonQuizReq {
    1:string userid,
    2:string lesson_id,
}

struct GetAllLessonQuizResp{
    1:common.resp resp,
}

service QuizService {
    CreateQuestionResp CreateQuestion(1: CreateQuestionReq req),
    TorFAnswerResp     TorFAnswer(1: TorFAnswerReq req),
    DelQuestionResp    DelQuestion(1: DelQuestionReq req),
    GetAllLessonQuizResp GetAllLessonQuiz(1: GetAllLessonQuizReq req),
}
