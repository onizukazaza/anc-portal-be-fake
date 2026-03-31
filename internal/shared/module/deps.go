package module

import (
	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/cache"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/localcache"
)

// Middleware holds reusable auth middleware handlers.
// Modules pick whichever auth strategy they need per route or group.
//
//	group.Get("/public", ctrl.Public)                          // no auth
//	group.Get("/me", deps.Middleware.JWTAuth, ctrl.Me)         // JWT
//	group.Get("/hook", deps.Middleware.APIKeyAuth, ctrl.Hook)  // API-Key
type Middleware struct {
	JWTAuth    fiber.Handler // Bearer-token (JWT) verification
	APIKeyAuth fiber.Handler // X-API-Key header verification
}

// Deps holds shared dependencies that modules receive during registration.
type Deps struct {
	Config      *config.Config
	DB          database.Provider
	Cache       cache.Cache
	LocalCache  localcache.Cache
	HybridCache *localcache.Hybrid
	Middleware  Middleware
}

// Module defines the interface every feature module must implement.
type Module interface {
	Register(router fiber.Router, deps Deps)
}
