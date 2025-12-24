package util

import (
	"math/rand"
	"time"
)

func GenerateRandomDigits(length int) string {
	rand.Seed(time.Now().UnixNano())
	digits := "0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = digits[rand.Intn(len(digits))]
	}
	return string(b)
}
