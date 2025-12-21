package util

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("empty password")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ComparePassword compares bcrypt hashed password with plaintext
func ComparePassword(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}

// HashToken returns a SHA256 hex of the token string for safe DB storage
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
