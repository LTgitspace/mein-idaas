package main

import (
	"log"
	"mein-idaas/middleware"
	"mein-idaas/seeder"
	"os"

	"github.com/gofiber/fiber/v2"
	swag "github.com/gofiber/swagger"
	"github.com/joho/godotenv"

	_ "mein-idaas/docs" // <-- required to register swagger spec

	"mein-idaas/controller"
	"mein-idaas/repository"
	"mein-idaas/service"
	"mein-idaas/util"
)

// @title           Mein IDaaS API
// @version         1.0
// @description     A custom Identification-as-a-Service server.
// @termsOfService  http://swagger.io/terms/

/// @contact.name    API Support
// @contact.email   support@swagger.io

// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html

// @host            localhost:4000
// @BasePath        /api/v1
func main() {
	// Load .env file with proper error handling
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: failed to load .env file: %v (using system environment variables)", err)
	}

	// Initialize Argon2 parameters from environment variables
	util.InitArgon2Params()

	// Initialize RSA keys for JWT signing
	if err := util.InitRSAKeys(); err != nil {
		log.Fatalf("failed to initialize RSA keys: %v", err)
	}

	db := util.InitDB()

	seeder.SeedRoles(db)

	userRepo := repository.NewUserRepository(db)
	credentialRepo := repository.NewCredentialRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	verificationRepo := repository.NewInMemoryVerificationRepo()

	util.StartDailyCleanup(refreshTokenRepo)
	emailService := service.NewEmailService()
	verificationService := service.NewVerificationService(verificationRepo, emailService)

	app := fiber.New()
	setupRoutes(app, userRepo, credentialRepo, refreshTokenRepo, roleRepo, verificationService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	log.Fatal(app.Listen(":" + port))
}

func setupRoutes(app *fiber.App, userRepo repository.UserRepository, credentialRepo repository.CredentialRepository, refreshTokenRepo repository.RefreshTokenRepository, roleRepo repository.RoleRepository, verificationService *service.VerificationService) {
	// Apply timer metrics middleware globally to all routes
	app.Use(middleware.TimerMetrics)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/swagger/*", swag.HandlerDefault)

	// create services and controllers
	authService := service.NewAuthService(userRepo, credentialRepo, refreshTokenRepo, roleRepo, verificationService)
	authController := controller.NewAuthController(authService)
	verifyController := controller.NewVerificationController(authService, verificationService)

	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	auth.Post("/register", authController.Register)
	auth.Post("/login", authController.Login)
	auth.Post("/refresh", authController.Refresh)

	// MFA endpoints
	auth.Post("/mfa/setup", authController.SetupMFA)
	auth.Get("/mfa/qrcode", authController.GetMFAQRCode)
	auth.Get("/mfa/qrcode/base64", authController.GetMFAQRCodeBase64)
	auth.Post("/mfa/confirm", authController.ConfirmMFA)

	// password change endpoints
	auth.Post("/password-change/send-otp", authController.SendPasswordChangeOTP)
	auth.Post("/password-change", authController.ChangePassword)

	// password reset endpoints (forgot password flow)
	auth.Post("/forgot-password/send-otp", authController.SendForgotPasswordOTP)
	auth.Post("/forgot-password/reset", authController.ResetPasswordWithOTP)

	// verification endpoints
	auth.Post("/verify", verifyController.VerifyEmail)
	auth.Post("/resend", verifyController.ResendVerificationCode)
}
