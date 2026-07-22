package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	newHashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return newHashedPassword, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	status, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	return status, nil
}

//
//func MakeJwt(userId uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
//
//}
