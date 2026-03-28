package auth

import (
	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/external"
	authhttp "github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/http"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
)

// Register wires auth module dependencies and mounts routes.
func Register(router fiber.Router, deps module.Deps) {
	userRepository := postgres.NewUserRepository(deps.DB.Main())
	tokenSigner := external.NewSimpleTokenSigner()
	authService := app.NewService(userRepository, tokenSigner)
	authController := authhttp.NewAuthController(authService)

	group := router.Group("/auth")
	group.Post("/login", authController.Login)
}
