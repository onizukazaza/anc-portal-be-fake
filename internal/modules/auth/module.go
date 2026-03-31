package auth

import (
	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/external"
	authhttp "github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/http"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// Register wires auth module dependencies and mounts routes.
// tokenSigner is created once by the caller and shared with auth middleware.
func Register(router fiber.Router, deps module.Deps, tokenSigner ports.TokenSigner) {
	userRepository := postgres.NewUserRepository(deps.DB.Main())
	authService := app.NewService(userRepository, tokenSigner)
	authController := authhttp.NewAuthController(authService)

	group := router.Group("/auth")
	group.Post("/login", authController.Login)
}

// NewTokenSigner creates the appropriate token signer based on stage configuration.
// Call once during server setup and share across middleware and auth module.
func NewTokenSigner(cfg *config.Config) ports.TokenSigner {
	if cfg.StageStatus == "local" {
		log.L().Warn().Msg("using SimpleTokenSigner (dev-only) — not for production")
		return external.NewSimpleTokenSigner()
	}
	return external.NewJWTTokenSigner(cfg.Server.JWTSecretKey, cfg.Server.JWTExpiry)
}
