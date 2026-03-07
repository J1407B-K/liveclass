namespace go common

struct resp {
    1: i16 Code
    2: string Msg
    3: optional data Data
}

union data {
    1: optional user userInfo
    2: optional lesson lessonInfo
    3: optional quiz quizInfo
    4: optional chat chatInfo

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
    6:i64 TeacherID
}

struct quiz {
    1:i64 QuizID
    2:i64 LessonID
    3:string Content
    4:i8 OptionsNum
    5:string Options
    6:string Answer
    7:i64 TeacherID
    8:list<quizStat> Stats
}

struct quizStat {
    1: string option
    2: i64 count
}

struct chat {
    1: string Message
}