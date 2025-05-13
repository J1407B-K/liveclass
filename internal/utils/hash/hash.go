package hash

import (
	"golang.org/x/crypto/bcrypt"
	"log"
)

func HashedLock(p string) (string, error) {
	hashedP, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return string(hashedP), nil
}

func CompareHashAndPassword(hashedP string, plainP string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedP), []byte(plainP))
}
