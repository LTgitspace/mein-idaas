package controller

import (
	"os"
	"time"

	"mein-idaas/dto"
	"mein-idaas/service"
	"mein-idaas/util"

	"github.com/gofiber/fiber/v2"
)

type VerificationController struct {
	authSvc         *service.AuthService
	verificationSvc *service.VerificationService
}

func NewVerificationController(authSvc *service.AuthService, verificationSvc *service.VerificationService) *VerificationController {
	return &VerificationController{
		authSvc:         authSvc,
		verificationSvc: verificationSvc,
	}
}

// VerifyEmail godoc
// @Summary      Verify email with OTP
// @Description  Verifies the 6-digit code sent to email. If successful, activates account and logs user in (returns tokens).
// @Tags         verification
// @Accept       json
// @Produce      json
// @Param        payload body dto.VerifyEmailRequest true "Verification payload"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /auth/verify [post]
func (vc *VerificationController) VerifyEmail(c *fiber.Ctx) error {
	// 1. Parse DTO
	var req dto.VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	// 2. Validate
	if err := util.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// 3. Get user by email first
	user, err := vc.authSvc.GetUserByEmail(req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	// 4. Verify the OTP code using user ID
	if err := vc.verificationSvc.VerifyCode(user.ID.String(), req.Code); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired verification code"})
	}

	// 5. Generate tokens
	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	// Extract Roles for Token
	var roleCodes []string
	for _, r := range user.Roles {
		roleCodes = append(roleCodes, r.Code)
	}

	// Generate Tokens with Roles
	pair, err := util.GenerateTokens(user.ID, roleCodes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate tokens"})
	}

	// Store refresh token in DB
	hash := util.HashToken(pair.RefreshToken)

	// Get refresh TTL from env (default 168h = 7 days)
	refreshTTLStr := os.Getenv("JWT_REFRESH_TTL")
	if refreshTTLStr == "" {
		refreshTTLStr = "168h"
	}
	refreshTTL, _ := time.ParseDuration(refreshTTLStr)

	if err := vc.authSvc.StoreRefreshToken(pair.RefreshID.String(), user.ID, hash, refreshTTL, clientIP, userAgent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store refresh token"})
	}

	// 6. Set Cookie
	cookiePath := os.Getenv("COOKIE_PATH")
	if cookiePath == "" {
		cookiePath = "/api/v1/auth"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    pair.RefreshToken,
		Expires:  time.Now().Add(refreshTTL),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     cookiePath,
	})

	// 7. Get access token TTL in seconds for response
	accessTTLStr := os.Getenv("JWT_ACCESS_TTL")
	if accessTTLStr == "" {
		accessTTLStr = "15m"
	}
	accessTTL, _ := time.ParseDuration(accessTTLStr)
	expiresIn := int(accessTTL.Seconds())

	// 8. Return Access Token
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": pair.AccessToken,
		"expires_in":   expiresIn,
	})
}
