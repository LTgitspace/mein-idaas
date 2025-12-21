package util

import (
	"errors"

	"mein-idaas/dto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ParseAccessToken now returns the full Claims object, not just ID
func ParseAccessToken(tokenString string) (*dto.AuthClaims, error) {
	claims := &dto.AuthClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return accessSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired access token")
	}

	return claims, nil
}

// ParseRefreshToken decodes and validates a refresh token, returning userID UUID and refresh jti UUID
func ParseRefreshToken(tokenString string) (uuid.UUID, uuid.UUID, error) {
	claims := &dto.AuthClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return refreshSecret, nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, uuid.Nil, errors.New("invalid or expired refresh token")
	}

	if claims.UserID == "" {
		return uuid.Nil, uuid.Nil, errors.New("missing user_id in refresh token")
	}

	if claims.ID == "" {
		return uuid.Nil, uuid.Nil, errors.New("missing jti in refresh token")
	}

	// Parse userID from string to UUID
	userID, err := uuid.Parse(claims.UserID)
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
