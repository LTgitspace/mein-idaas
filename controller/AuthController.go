package controller

import (
	"os"
	"time"

	"mein-idaas/dto"
	"mein-idaas/service"
	"mein-idaas/util"

	"github.com/gofiber/fiber/v2"
)

// AuthController provides handlers for authentication
type AuthController struct {
	svc *service.AuthService
}

func NewAuthController(s *service.AuthService) *AuthController {
	return &AuthController{svc: s}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a user account with email and password. Assigns default 'user' role.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body dto.RegisterRequest true "Register payload"
// @Success      201  {object}  dto.RegisterResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/register [post]
func (ac *AuthController) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	res, err := ac.svc.Register(&req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// Login godoc
// @Summary      Login with email and password
// @Description  Validates credentials, returns Access Token in JSON, and sets Refresh Token in HttpOnly Cookie. If email is not verified, sends verification email and returns 403.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body dto.LoginRequest true "Login payload"
// @Success      200  {object}  map[string]interface{} "Returns {access_token, expires_in}"
// @Header       200  {string}  Set-Cookie "refresh_token=...; HttpOnly; Secure"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string "Email not verified - verification email sent"
// @Failure      500  {object}  map[string]string
// @Router       /auth/login [post]
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	res, err := ac.svc.Login(&req, clientIP, userAgent)
	if err != nil {
		if err.Error() == "invalid credentials" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		if err.Error() == "email not verified" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "email not verified", "message": "verification email has been sent to your email address"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Get refresh token TTL from env, default to 168h (7 days)
	refreshTTL := os.Getenv("JWT_REFRESH_TTL")
	if refreshTTL == "" {
		refreshTTL = "168h"
	}
	duration, _ := time.ParseDuration(refreshTTL)

	// Get cookie path from env, default to /api/v1/auth
	cookiePath := os.Getenv("COOKIE_PATH")
	if cookiePath == "" {
		cookiePath = "/api/v1/auth"
	}

	// SECURE COOKIE SETTING
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		Expires:  time.Now().Add(duration),
		HTTPOnly: true,     // JS cannot access
		Secure:   true,     // HTTPS only (set false for localhost if needed)
		SameSite: "Strict", // CSRF protection
		Path:     cookiePath,
	})

	// Return only Access Token to client memory
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":  res.AccessToken,
		"refresh_token": res.RefreshToken, //remove after production
		"expires_in":    res.ExpiresIn,
	})
}

// Refresh godoc
// @Summary      Rotate refresh token
// @Description  Reads 'refresh_token' from HttpOnly Cookie and issues a new Access/Refresh pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Cookie header string false "Cookie containing refresh_token"
// @Success      200  {object}  map[string]interface{} "Returns {access_token, expires_in}"
// @Header       200  {string}  Set-Cookie "refresh_token=...; HttpOnly; Secure"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/refresh [post]
func (ac *AuthController) Refresh(c *fiber.Ctx) error {
	// 1. Get Token from Cookie
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing refresh token cookie"})
	}

	// 2. Prepare request
	req := dto.RefreshRequest{
		RefreshToken: refreshToken,
	}

	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	// 3. Call Service
	res, err := ac.svc.Refresh(&req, clientIP, userAgent)
	if err != nil {
		// Clear cookie on failure
		c.ClearCookie("refresh_token")

		if err.Error() == "invalid or unknown refresh token" || err.Error() == "refresh token expired or revoked" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Get refresh token TTL from env, default to 168h (7 days)
	refreshTTL := os.Getenv("JWT_REFRESH_TTL")
	if refreshTTL == "" {
		refreshTTL = "168h"
	}
	duration, _ := time.ParseDuration(refreshTTL)

	// Get cookie path from env, default to /api/v1/auth
	cookiePath := os.Getenv("COOKIE_PATH")
	if cookiePath == "" {
		cookiePath = "/api/v1/auth"
	}

	// 4. Rotate Cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		Expires:  time.Now().Add(duration),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     cookiePath,
	})

	// 5. Return new Access Token
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": res.AccessToken,
		"expires_in":   res.ExpiresIn,
	})
}

// SendPasswordChangeOTP godoc
// @Summary      Send OTP for password change
// @Description  Sends a 6-digit OTP code to the authenticated user's email for password change verification. Requires valid access token in Authorization header.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer <access_token>"
// @Success      200  {object}  dto.PasswordChangeSendOTPResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/password-change/send-otp [post]
func (ac *AuthController) SendPasswordChangeOTP(c *fiber.Ctx) error {
	// 1. Extract user ID from Authorization header (JWT token)
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
	}

	// Parse Bearer token to get user ID
	userID, err := util.ExtractUserIDFromToken(authHeader)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
	}

	// 2. Send OTP using user ID
	userEmail, err := ac.svc.SendPasswordChangeOTPByUserID(userID)
	if err != nil {
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(dto.PasswordChangeSendOTPResponse{
		Message: "OTP sent to your email",
		Email:   userEmail,
	})
}

// ChangePassword godoc
// @Summary      Change password with OTP verification
// @Description  Changes the user's password. Requires old password, new password, and OTP code. User ID is read from JWT access token header.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer <access_token>"
// @Param        payload body dto.PasswordChangeRequest true "Password change payload"
// @Success      200  {object}  dto.PasswordChangeResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/password-change [post]
func (ac *AuthController) ChangePassword(c *fiber.Ctx) error {
	// 1. Extract user ID from Authorization header (JWT token)
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
	}

	// Parse Bearer token
	userID, err := util.ExtractUserIDFromToken(authHeader)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
	}

	// 2. Parse request body
	var req dto.PasswordChangeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	// 3. Validate request
	if err := util.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// 4. Check if old and new passwords are the same
	if req.OldPassword == req.NewPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "new password must be different from old password"})
	}

	// 5. Call service to change password
	if err := ac.svc.ChangePassword(userID, req.OldPassword, req.NewPassword, req.OTPCode); err != nil {
		if err.Error() == "invalid old password" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid old password"})
		}
		if err.Error() == "invalid verification code" || err.Error() == "code expired" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Get user to return email
	user, err := ac.svc.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch user"})
	}

	return c.Status(fiber.StatusOK).JSON(dto.PasswordChangeResponse{
		Message: "password changed successfully",
		Email:   user.Email,
	})
}
