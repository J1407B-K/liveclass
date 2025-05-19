package model

type Answer struct {
	ID              int         `json:"answer_id" gorm:"primary_key;auto_increment"`
	Right           string      `json:"right" gorm:"not null"`
	QuestionId      int         `json:"question_id" gorm:"unique;not null"`
	OptionNums      int         `json:"option_nums" gorm:"not null"`
	SelectedOptions StringArray `json:"selected_options" gorm:"type:json"`
	AnsweredId      StringArray `json:"answered_id" gorm:"type:json"`
}
