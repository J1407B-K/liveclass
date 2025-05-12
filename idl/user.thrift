namespace go user

include "common.thrift"

struct RegisterReq{
    1: string username,
    2: string password,
    3: string auth,
    4: string lessons,
}

struct RegisterResp{
    1: common.resp resp
}

struct LoginReq{
    1: string username,
    2: string password,
    3: string auth
}

struct LoginResp{
    1: common.resp resp
}

struct CreateLessonReq{
    1: string name,
}

struct CreateLessonResp{
    1: common.resp resp,
}

struct GetUserInfoReq{
    1: string username,
}

struct GetUserInfoResp{
    1:  common.resp resp,
}

struct AddStudentReq{
    1: string lesson,
    2: string name,
}

struct AddStudentResp{
    1: common.resp resp,
}

struct DeleteStudentReq{
    1: string lesson,
    2: string name,
}

struct DeleteStudentResp{
    1: common.resp resp
}

service UserService{
    RegisterResp Register(1: RegisterReq req),
    LoginResp Login(1: LoginReq req),
    CreateLessonResp CreateLesson(1: CreateLessonReq req),
    AddStudentResp AddStudent(1: AddStudentReq req),
    DeleteStudentResp DeleteStudent(1: DeleteStudentReq req),
    GetUserInfoResp  GetUserInfo(1: GetUserInfoReq req)
}