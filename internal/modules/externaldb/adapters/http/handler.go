package http

import (
	"github.com/gofiber/fiber/v2"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb/ports"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/enum"
)

type Handler struct {
	service ports.ExternalDBService
}

func NewHandler(service ports.ExternalDBService) *Handler {
	return &Handler{service: service}
}

func NewExternalDBController(service ports.ExternalDBService) ports.ExternalDBController {
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
// @Failure 400 {object} dto.ErrorResponse "[13001] extdb-name-required — ไม่ได้ส่ง database name"
// @Failure 404 {object} dto.ErrorResponse "[13002] extdb-not-found — ไม่พบ database ที่ระบุ"
// @Failure 503 {object} dto.ErrorResponse "[13003] extdb-unhealthy — database ไม่สามารถเชื่อมต่อได้"
// @Security BearerAuth
// @Router /external-db/health/{name} [get]
func (h *Handler) CheckByName(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerExtDBHandler).Start(c.UserContext(), "CheckByName")
	defer span.End()

	name := c.Params("name")
	if name == "" {
		return dto.ErrorWithTrace(c, fiber.StatusBadRequest, "database name is required", dto.TraceExtDBNameRequired)
	}

	result := h.service.CheckByName(ctx, name)
	if result.Status == enum.DBError {
		return dto.ErrorWithTrace(c, fiber.StatusNotFound, result.Error, dto.TraceExtDBNotFound)
	}
	if result.Status == enum.DBUnhealthy {
		return dto.ErrorWithTrace(c, fiber.StatusServiceUnavailable, result.Error, dto.TraceExtDBUnhealthy)
	}

	return dto.Success(c, fiber.StatusOK, result)
}
