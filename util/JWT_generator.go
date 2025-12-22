package util

import (
	"mein-idaas/dto"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	RefreshID    uuid.UUID
}

// Load keys once at startup
var (
	accessSecret  = []byte(getEnv("JWT_ACCESS_SECRET", "fallback-dev-secret"))
	refreshSecret = []byte(getEnv("JWT_REFRESH_SECRET", "fallback-dev-secret"))
)

// GenerateTokens creates both Access and Refresh tokens
func GenerateTokens(userID uuid.UUID, roles []string) (*TokenPair, error) {
	now := time.Now()

	// 1. Create Access Token
	accessClaims := dto.AuthClaims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "my-idaas",
			Audience:  jwt.ClaimStrings{"my-game-server", "smoking-app"},
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString(accessSecret)
	if err != nil {
		return nil, err
	}

	// 2. Create Refresh Token
	refreshID := uuid.New()
	refreshClaims := dto.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "my-idaas",
			ID:        refreshID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString(refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  signedAccess,
		RefreshToken: signedRefresh,
		RefreshID:    refreshID,
	}, nil
}

// SignRefreshToken creates a JWT string for an EXISTING refresh token ID
func SignRefreshToken(refreshID uuid.UUID, userID uuid.UUID) (string, error) {
	now := time.Now()

	claims := dto.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "my-idaas",
			ID:        refreshID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(refreshSecret)
}

// GenerateAccessTokenOnly creates a short-lived JWT for the user.
// Used specifically in Refresh Token Rotation (Grace Period).
func GenerateAccessTokenOnly(userID uuid.UUID, roles []string) (string, error) {
	now := time.Now()

	// Use dto.AuthClaims to ensure this token looks EXACTLY like a normal login token
	claims := dto.AuthClaims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "my-idaas",
			Audience:  jwt.ClaimStrings{"my-game-server", "smoking-app"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// ✅ FIX: Use 'accessSecret' instead of undefined 'secretKey'
	return token.SignedString(accessSecret)
}
