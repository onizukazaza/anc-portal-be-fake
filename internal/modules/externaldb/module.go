package externaldb

import (
	"github.com/gofiber/fiber/v2"

	exthttp "github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb/adapters/http"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
)

// Register wires external-db module dependencies and mounts routes.
func Register(router fiber.Router, deps module.Deps) {
	names := make([]string, 0, len(deps.Config.ExternalDBs))
	for name := range deps.Config.ExternalDBs {
		names = append(names, name)
	}

	service := app.NewService(deps.DB, names)
	controller := exthttp.NewExternalDBController(service)

	group := router.Group("/external-db")
	group.Get("/health", controller.CheckAll)
	group.Get("/health/:name", controller.CheckByName)
}
