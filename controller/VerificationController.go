package controller

import (
	"log"

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
// @Description  Verifies the 6-digit code sent to email. If successful, activates account (sets isEmailVerified=true).
// @Tags         verification
// @Accept       json
// @Produce      json
// @Param        payload body dto.VerifyEmailRequest true "Verification payload"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
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

	// 5. Mark user as verified
	if err := vc.authSvc.MarkEmailVerified(user.ID.String()); err != nil {
		log.Printf("Failed to mark email verified for %s: %v", req.Email, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update user verification status"})
	}
	log.Printf("Email verified for %s (user_id=%s)", req.Email, user.ID.String())

	// 6. Return success (token generation and storage removed)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "email verified"})
}

// ResendVerificationCode godoc
// @Summary      Resend verification code to email
// @Description  Generates and sends a new verification code to the specified email if the user exists.
// @Tags         verification
// @Accept       json
// @Produce      json
// @Param        payload body dto.ResendOTPRequest true "Resend payload"
// @Success      202  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/resend [post]
func (vc *VerificationController) ResendVerificationCode(c *fiber.Ctx) error {
	var req dto.ResendOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if err := util.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := vc.authSvc.GetUserByEmail(req.Email)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	if err := vc.verificationSvc.SendVerificationCode(user.ID.String(), user.Email); err != nil {
		log.Printf("Failed to initiate verification email for %s: %v", req.Email, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send verification code"})
	}

	log.Printf("Verification code send initiated for %s", req.Email)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"message": "verification code sent"})
}
