package util

import (
	"strings"
)

// IsDuplicateKeyError checks if the error is a database constraint violation
func IsDuplicateKeyError(err error) bool {
	// This string check works for Postgres "SQLSTATE 23505"
	return strings.Contains(err.Error(), "duplicate key value") ||
		strings.Contains(err.Error(), "23505")
}
