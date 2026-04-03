package auth

import (
	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/adapters/external"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// NewTokenSigner creates the appropriate token signer based on stage configuration.
// Call once during server setup and share across middleware and auth module.
func NewTokenSigner(cfg *config.Config) ports.TokenSigner {
	if cfg.StageStatus == "local" {
		log.L().Warn().Msg("using SimpleTokenSigner (dev-only) — not for production")
		return external.NewSimpleTokenSigner()
	}
	return external.NewJWTTokenSigner(cfg.Server.JWTSecretKey, cfg.Server.JWTExpiry)
}

// TODO: implement Register() to wire auth login/signup routes
// func Register(router fiber.Router, deps module.Deps, tokenSigner ports.TokenSigner) {
//     userRepo := postgres.NewUserRepository(deps.DB.Main())
//     authService := app.NewService(userRepo, tokenSigner)
//     authController := authhttp.NewAuthController(authService)
//     group := router.Group("/auth")
//     group.Post("/login", authController.Login)
// }
