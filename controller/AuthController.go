package controller

import (
	"time"

	"mein-idaas/dto"
	"mein-idaas/service"

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
// @Description  Validates credentials, returns Access Token in JSON, and sets Refresh Token in HttpOnly Cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload body dto.LoginRequest true "Login payload"
// @Success      200  {object}  map[string]interface{} "Returns {access_token, expires_in}"
// @Header       200  {string}  Set-Cookie "refresh_token=...; HttpOnly; Secure"
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// SECURE COOKIE SETTING
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour), // 7 days
		HTTPOnly: true,                               // JS cannot access
		Secure:   true,                               // HTTPS only (set false for localhost if needed)
		SameSite: "Strict",                           // CSRF protection
		Path:     "/api/v1/auth",                     // Only sent to auth endpoints
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

	// 4. Rotate Cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/api/v1/auth",
	})

	// 5. Return new Access Token
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": res.AccessToken,
		"expires_in":   res.ExpiresIn,
	})
}
