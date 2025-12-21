package controller

import (
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
// @Summary Register a new user
// @Description Create a user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body dto.RegisterRequest true "Register payload"
// @Success 201 {object} dto.RegisterResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/register [post]
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
// @Summary Login with email and password
// @Description Returns access and refresh token pair
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body dto.LoginRequest true "Login payload"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/login [post]
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	res, err := ac.svc.Login(&req, clientIP, userAgent)
	if err != nil {
		// map common auth errors to 401
		if err.Error() == "invalid credentials" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// Refresh godoc
// @Summary Rotate refresh token and issue new access token
// @Description Exchange a refresh token for a new token pair (rotation)
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body dto.RefreshRequest true "Refresh payload"
// @Success 200 {object} dto.RefreshResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/refresh [post]
func (ac *AuthController) Refresh(c *fiber.Ctx) error {
	var req dto.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	clientIP := c.IP()
	userAgent := c.Get("User-Agent")

	res, err := ac.svc.Refresh(&req, clientIP, userAgent)
	if err != nil {
		if err.Error() == "invalid or unknown refresh token" || err.Error() == "refresh token expired or revoked" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(res)
}
