package util

import "os"

// getEnv retrieves an environment variable or returns a fallback
// It is available to ALL files in package 'util'
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
