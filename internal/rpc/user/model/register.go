package model

type RegisterParam struct {
	Username     string
	PasswordHash string
	Auth         string
	Status       int8
}
