package module

import (
	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/cache"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/localcache"
)

// Deps holds shared dependencies that modules receive during registration.
type Deps struct {
	Config      *config.Config
	DB          database.Provider
	Cache       cache.Cache
	LocalCache  localcache.Cache
	HybridCache *localcache.Hybrid
}

// Module defines the interface every feature module must implement.
type Module interface {
	Register(router fiber.Router, deps Deps)
}
