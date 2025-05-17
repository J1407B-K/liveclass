namespace go quiz

include "common.thrift"

struct CreateQuestionReq {
    1: string LessonName,
    2: string Userid,
    3: string Content,
    4: list<string> Options,
    5: string Answer,
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

struct ListAllUserQuizReq {
    1: string Userid
}

struct ListAllUserQuizResp {
    1: common.resp resp,
}

service QuizService {
    CreateQuestionResp CreateQuestion(1: CreateQuestionReq req),
    TorFAnswerResp     TorFAnswer(1: TorFAnswerReq req),
    ListAllUserQuizResp  ListAllUserQuiz(1:ListAllUserQuizReq req),
}
