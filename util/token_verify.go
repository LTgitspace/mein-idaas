package util

import (
	"errors"

	"mein-idaas/dto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ParseAccessToken validates and returns the access token claims using RS256
func ParseAccessToken(tokenString string) (*dto.AuthClaims, error) {
	claims := &dto.AuthClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify RS256 signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("invalid signing method, expected RS256")
		}
		return GetPublicKey(), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired access token")
	}

	return claims, nil
}

// ParseRefreshToken decodes and validates a refresh token using RS256
func ParseRefreshToken(tokenString string) (uuid.UUID, uuid.UUID, error) {
	claims := &dto.AuthClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify RS256 signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("invalid signing method, expected RS256")
		}
		return GetPublicKey(), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, uuid.Nil, errors.New("invalid or expired refresh token")
	}

	// Use standard 'Subject' claim
	if claims.Subject == "" {
		return uuid.Nil, uuid.Nil, errors.New("missing subject (user_id) in refresh token")
	}

	if claims.ID == "" {
		return uuid.Nil, uuid.Nil, errors.New("missing jti in refresh token")
	}

	// Parse userID from string to UUID
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("invalid user_id format in token")
	}

	// Parse jti (refreshID) from string to UUID
	refreshID, err := uuid.Parse(claims.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("invalid jti format in token")
	}

	return userID, refreshID, nil
}
