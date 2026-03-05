package model

type Question struct {
	LessonId   int64    `json:"lesson_id"`
	Content    string   `json:"content"`
	OptionNums int      `json:"option_nums"`
	Options    []string `json:"options"`
	Answer     string   `json:"answer"`
	Duration   int32    `json:"duration"`
}

type Answer struct {
	QuestionId int64  `json:"question_id"`
	Answer     string `json:"answer"`
}
