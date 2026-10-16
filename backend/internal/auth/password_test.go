package auth

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"testing"
)

func TestHashPasswordRejectsShortPasswords(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "empty", password: "", wantErr: true},
		{name: "one character", password: "a", wantErr: true},
		{name: "seven characters", password: "1234567", wantErr: true},
		{name: "exactly eight characters", password: "12345678"},
		{name: "long password", password: strings.Repeat("correct horse ", 4)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashPassword(tc.password)
			if tc.wantErr {
				if !errors.Is(err, ErrPasswordTooShort) {
					t.Fatalf("HashPassword() error = %v, want ErrPasswordTooShort", err)
				}
				if hash != "" {
					t.Error("HashPassword() returned a hash alongside the error, want an empty string")
				}
				return
			}
			if err != nil {
				t.Fatalf("HashPassword() returned an unexpected error: %v", err)
			}
			if hash == tc.password {
				t.Error("HashPassword() returned the password unchanged")
			}
			if cost, err := bcrypt.Cost([]byte(hash)); err != nil {
				t.Errorf("the hash is not a valid bcrypt hash: %v", err)
			} else if cost != bcryptCost {
				t.Errorf("bcrypt cost = %d, want %d", cost, bcryptCost)
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	const password = "Sust-250-project"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name      string
		hash      string
		candidate string
		wantErr   bool
	}{
		{name: "correct password", hash: hash, candidate: password},
		{name: "wrong password", hash: hash, candidate: "Sust-250-projecT", wantErr: true},
		{name: "empty password", hash: hash, candidate: "", wantErr: true},
		{name: "password with trailing space", hash: hash, candidate: password + " ", wantErr: true},
		{name: "hash is not a bcrypt hash", hash: "plain-text", candidate: password, wantErr: true},
		{name: "empty hash", hash: "", candidate: password, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckPassword(tc.hash, tc.candidate)
			if tc.wantErr {
				if !errors.Is(err, ErrWrongPassword) {
					t.Errorf("CheckPassword() error = %v, want ErrWrongPassword", err)
				}
				return
			}
			if err != nil {
				t.Errorf("CheckPassword() returned an unexpected error: %v", err)
			}
		})
	}
}

func TestHashPasswordUsesASaltPerCall(t *testing.T) {
	const password = "same-password-twice"
	first, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() returned an unexpected error: %v", err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() returned an unexpected error: %v", err)
	}
	if first == second {
		t.Error("hashing the same password twice produced the same hash, want a fresh salt each time")
	}
	for _, hash := range []string{first, second} {
		if err := CheckPassword(hash, password); err != nil {
			t.Errorf("CheckPassword() rejected a hash of the correct password: %v", err)
		}
	}
}
