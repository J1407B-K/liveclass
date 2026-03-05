namespace go common

struct resp {
    1: i16 Code
    2: string Msg
    3: optional data Data
}

union data {
    1: optional user userInfo
    2: optional lesson lessonInfo

    10: string sdp
    11: string filename
    12: list<i64> idList
    13: string text
}

struct user {
    1: i64 UserID
    2: string UserName
    3: string Auth
    4: i8 Status
}

struct lesson {
    1:i64 LessonID
    2:string Name
    3:string TeacherName
    4:string Description
    5:list<i64> StudentID
}
