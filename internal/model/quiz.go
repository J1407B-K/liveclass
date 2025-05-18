package model

type Question struct {
	LessonId string   `json:"lesson_id"`
	Content  string   `json:"content"`
	Options  []string `json:"options"`
	Answer   string   `json:"answer"`
}

type Answer struct {
	QuestionId string `json:"question_id"`
	Answer     string `json:"answer"`
}
