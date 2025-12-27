package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2 parameters for password hashing
const (
	argon2Time      = 3
	argon2Memory    = 64 * 1024 // 64 MB
	argon2Threads   = 4
	argon2KeyLength = 32
	argon2SaltLen   = 16
)

// HashPassword hashes a plaintext password using argon2
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("empty password")
	}

	// Generate random salt
	salt, err := generateSalt(argon2SaltLen)
	if err != nil {
		return "", err
	}

	// Hash password with argon2
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLength)

	// Combine salt and hash for storage
	return encodeArgon2Hash(salt, hash), nil
}

// ComparePassword compares argon2 hashed password with plaintext
func ComparePassword(hashed, plain string) error {
	salt, hash, err := decodeArgon2Hash(hashed)
	if err != nil {
		return err
	}

	// Hash the provided password with the same salt
	computedHash := argon2.IDKey([]byte(plain), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLength)

	// Compare hashes
	if !constantTimeCompare(hash, computedHash) {
		return errors.New("invalid password")
	}

	return nil
}

// HashToken returns a SHA256 hex of the token string for safe DB storage
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// generateSalt generates a random salt for argon2
func generateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// encodeArgon2Hash encodes salt and hash into a string format
func encodeArgon2Hash(salt, hash []byte) string {
	return "$argon2id$v=19$m=65536,t=3,p=4$" +
		hex.EncodeToString(salt) + "$" +
		hex.EncodeToString(hash)
}

// decodeArgon2Hash decodes the argon2 hash format back to salt and hash
func decodeArgon2Hash(encoded string) ([]byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, errors.New("invalid argon2 hash format")
	}

	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return nil, nil, errors.New("invalid salt encoding")
	}

	hash, err := hex.DecodeString(parts[5])
	if err != nil {
		return nil, nil, errors.New("invalid hash encoding")
	}

	return salt, hash, nil
}

// constantTimeCompare compares two byte slices in constant time
func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := 0
	for i := range a {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}
