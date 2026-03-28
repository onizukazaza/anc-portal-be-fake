package http

import (
	"github.com/gofiber/fiber/v2"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/enum"
)

type Handler struct {
	service *app.Service
}

func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

func NewExternalDBController(service *app.Service) ExternalDBController {
	return &Handler{service: service}
}

// CheckAll godoc
// @Summary Check all external databases
// @Description Test connectivity of all registered external databases
// @Tags ExternalDB
// @Accept json
// @Produce json
// @Success 200 {object} dto.ApiResponse "All database statuses"
// @Security BearerAuth
// @Router /external-db/health [get]
func (h *Handler) CheckAll(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerExtDBHandler).Start(c.UserContext(), "CheckAll")
	defer span.End()

	results := h.service.CheckAll(ctx)
	return dto.Success(c, fiber.StatusOK, results)
}

// CheckByName godoc
// @Summary Check external database by name
// @Description Test connectivity of a specific external database
// @Tags ExternalDB
// @Accept json
// @Produce json
// @Param name path string true "Database name"
// @Success 200 {object} dto.ApiResponse "Database status"
// @Failure 400 {object} dto.ApiResponse "Name is required"
// @Failure 404 {object} dto.ApiResponse "Database not found"
// @Security BearerAuth
// @Router /external-db/health/{name} [get]
func (h *Handler) CheckByName(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerExtDBHandler).Start(c.UserContext(), "CheckByName")
	defer span.End()

	name := c.Params("name")
	if name == "" {
		return dto.Error(c, fiber.StatusBadRequest, "database name is required")
	}

	result := h.service.CheckByName(ctx, name)
	if result.Status == enum.DBError {
		return dto.Error(c, fiber.StatusNotFound, result.Error)
	}

	return dto.Success(c, fiber.StatusOK, result)
}
