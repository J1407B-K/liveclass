package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultCost = bcrypt.DefaultCost
)

func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("password is empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), defaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func VerifyPassword(input string, hash string) (bool, error) {
	if input == "" || hash == "" {
		return false, errors.New("input or hash is empty")
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input))
	if err != nil {
		return false, err
	}
	return true, nil
}
