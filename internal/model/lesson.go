package model

type Lesson struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Teacher     string `json:"teacher"`
	Code        string `json:"code"`
}
