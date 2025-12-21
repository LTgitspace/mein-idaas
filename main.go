package main

import (
	"log"
	"mein-idaas/seeder"

	"github.com/gofiber/fiber/v2"
	swag "github.com/gofiber/swagger"

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

// @contact.name    API Support
// @contact.email   support@swagger.io

// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html

// @host            localhost:4000
// @BasePath        /api/v1
func main() {
	db := util.InitDB()

	seeder.SeedRoles(db)

	userRepo := repository.NewUserRepository(db)
	credentialRepo := repository.NewCredentialRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	roleRepo := repository.NewRoleRepository(db)

	app := fiber.New()
	setupRoutes(app, userRepo, credentialRepo, refreshTokenRepo, roleRepo)

	log.Fatal(app.Listen(":4000"))
}

func setupRoutes(app *fiber.App, userRepo repository.UserRepository, credentialRepo repository.CredentialRepository, refreshTokenRepo repository.RefreshTokenRepository, roleRepo repository.RoleRepository) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/swagger/*", swag.HandlerDefault)

	authService := service.NewAuthService(userRepo, credentialRepo, refreshTokenRepo, roleRepo)
	authController := controller.NewAuthController(authService)

	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	auth.Post("/register", authController.Register)
	auth.Post("/login", authController.Login)
	auth.Post("/refresh", authController.Refresh)
}
