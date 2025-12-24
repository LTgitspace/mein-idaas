package repository

import (
	"time"
)

type VerificationRepository interface {
	// Save stores the code with a strict TTL
	Save(key string, code string, duration time.Duration) error

	// Get retrieves the code. Returns error if expired or not found.
	Get(key string) (string, error)

	// Delete removes the code (used after successful verification)
	Delete(key string) error
}
