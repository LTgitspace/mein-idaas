package util

import (
	"log"
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

// Load keys and TTLs once at startup
var (
	accessTTL  = parseTokenTTL("JWT_ACCESS_TTL", 15*time.Minute)
	refreshTTL = parseTokenTTL("JWT_REFRESH_TTL", 168*time.Hour)
	issuer     = getEnv("JWT_ISSUER", "mein-idaas")
)

// parseTokenTTL parses a duration from env variable or returns default
func parseTokenTTL(envKey string, defaultDuration time.Duration) time.Duration {
	ttlStr := getEnv(envKey, "")
	if ttlStr == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(ttlStr)
	if err != nil {
		log.Printf("warning: invalid %s value '%s', using default %v\n", envKey, ttlStr, defaultDuration)
		return defaultDuration
	}
	return duration
}

// GenerateTokens creates both Access and Refresh tokens using RS256
func GenerateTokens(userID uuid.UUID, roles []string) (*TokenPair, error) {
	now := time.Now()

	// 1. Create Access Token
	accessClaims := dto.AuthClaims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{"self-hosted-idaas"},
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	signedAccess, err := accessToken.SignedString(GetPrivateKey())
	if err != nil {
		return nil, err
	}

	// 2. Create Refresh Token
	refreshID := uuid.New()
	refreshClaims := dto.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    issuer,
			ID:        refreshID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString(GetPrivateKey())
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
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    issuer,
			ID:        refreshID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(GetPrivateKey())
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
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{"my-game-server", "smoking-app"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(GetPrivateKey())
}
