package dto

import (
	"github.com/golang-jwt/jwt/v5"
)

// AuthClaims will be encoded inside the token
type AuthClaims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	// Standard claims (exp, iss, iat) are embedded here
	jwt.RegisteredClaims
}
