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
	RefreshID    uuid.UUID // We return this so you can save it to DB
}

// Load keys once at startup to prevent reading env on every request
var (
	accessSecret  = []byte(getEnv("JWT_ACCESS_SECRET", "fallback-dev-secret"))
	refreshSecret = []byte(getEnv("JWT_REFRESH_SECRET", "fallback-dev-secret"))
)

// GenerateTokens now accepts ROLES to bake them into the token
func GenerateTokens(userID uuid.UUID, roles []string) (*TokenPair, error) {
	now := time.Now()

	// 1. Create Access Token
	accessClaims := dto.AuthClaims{
		Roles: roles, // <--- INJECT ROLES HERE
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "my-idaas",
			// Audience is important for IDaaS (who is this token for?)
			Audience: jwt.ClaimStrings{"my-game-server", "smoking-app"},
		},
	}

	// TODO: For IDaaS, switch to jwt.SigningMethodRS256 and use a Private Key
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := accessToken.SignedString(accessSecret)
	if err != nil {
		return nil, err
	}

	// 2. Create Refresh Token (Opaque to the user, strictly for the IDaaS)
	refreshID := uuid.New()
	refreshClaims := dto.AuthClaims{
		// No roles needed in refresh token usually
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

//func getEnv(key, fallback string) string {
//	if value, exists := os.LookupEnv(key); exists {
//		return value
//	}
//	return fallback
//}
