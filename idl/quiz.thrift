namespace go quiz

include "common.thrift"

struct CreateQuestionReq {
    1: i64 LessonId,
    2: i64 Userid,
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
    1: i64 Question_id,
    2: i64 Userid,
    3: string UserAnswer,
}

struct TorFAnswerResp {
    1: common.resp resp,
}

struct DelQuestionReq {
    1:i64 Userid,
    2:i64 Question_id,
}

struct DelQuestionResp {
    1: common.resp resp,
}

struct GetAllLessonQuizReq {
    1:i64 userid,
    2:i64 lesson_id,
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
