package example

import (
	"github.com/gofiber/fiber/v2"

	examplehttp "github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/adapters/http"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/adapters/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
)

// Register wires Example module dependencies and mounts routes.
// This is the Composition Root — the only place where concrete types are assembled.
func Register(router fiber.Router, deps module.Deps) {
	// >> 1. Create adapters (concrete implementations)
	repo := postgres.NewRepository(deps.DB.Main())

	// >> 2. Create service (inject ports/interfaces)
	svc := app.NewService(repo)

	// >> 3. Create controller (HTTP transport)
	ctrl := examplehttp.NewController(svc)

	// >> 4. Mount routes (apply auth middleware as needed)
	group := router.Group("/examples")
	group.Get("/", ctrl.List)                                // public
	group.Get("/:id", deps.Middleware.JWTAuth, ctrl.GetByID) // protected
}
