package auth

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost        = 12
	MinPasswordLength = 8
)

var (
	ErrPasswordTooShort = fmt.Errorf("auth: password must be at least %d characters long", MinPasswordLength)
	ErrWrongPassword    = errors.New("auth: the password does not match")
)

func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: could not hash the password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrWrongPassword
	}
	return nil
}
