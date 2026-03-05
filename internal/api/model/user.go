package model

type User struct {
	UserId int64 `json:"userId"`

	Username string `json:"username"`
	Password string `json:"password"`

	Auth   string `json:"auth"`
	Status int8   `json:"status"`
}
