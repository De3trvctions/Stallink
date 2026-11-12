package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	if len(password) < 8 || len(password) > 72 {
		return "", fmt.Errorf("auth: password length")
	}
	sum, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(sum), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func NewToken() (plain string, hash []byte, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", nil, fmt.Errorf("session token: %w", err)
	}
	sum := sha256.Sum256(buf)
	return base64.RawURLEncoding.EncodeToString(buf), sum[:], nil
}

func HashToken(plain string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(plain)
	if err != nil || len(raw) != 32 {
		return nil, fmt.Errorf("session token: invalid")
	}
	sum := sha256.Sum256(raw)
	return sum[:], nil
}
