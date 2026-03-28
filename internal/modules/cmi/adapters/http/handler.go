package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

type Handler struct {
	service *app.Service
}

func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

func NewCMIController(service *app.Service) CMIController {
	return &Handler{service: service}
}

// GetPolicyByJobID godoc
// @Summary      Get CMI policy by job ID
// @Description  ดึงข้อมูลงาน พรบ. เดี่ยว (Compulsory Motor Insurance) ตาม job_id
// @Tags         CMI
// @Accept       json
// @Produce      json
// @Param        job_id path string true "Job ID"
// @Success      200 {object} dto.ApiResponse "CMI policy data"
// @Failure      400 {object} dto.ApiResponse "Job ID is required"
// @Failure      404 {object} dto.ApiResponse "Job not found"
// @Failure      500 {object} dto.ApiResponse "Internal error"
// @Security     BearerAuth
// @Router       /cmi/{job_id}/request-policy-single-cmi [get]
func (h *Handler) GetPolicyByJobID(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerCMIHandler).Start(c.UserContext(), "GetPolicyByJobID")
	defer span.End()
	start := time.Now()

	jobID := c.Params("job_id")
	span.SetAttributes(attribute.String("job_id", jobID))
	log.L().Info().Str("layer", "handler").Str("job_id", jobID).Msg("→ CMI GetPolicyByJobID")

	if jobID == "" {
		return dto.Error(c, fiber.StatusBadRequest, "job_id is required")
	}

	policy, err := h.service.GetPolicyByJobID(ctx, jobID)
	if err != nil {
		elapsed := time.Since(start)
		if errors.Is(err, app.ErrJobNotFound) {
			log.L().Warn().Str("layer", "handler").Str("job_id", jobID).Dur("elapsed", elapsed).Msg("← CMI job not found")
			return dto.Error(c, fiber.StatusNotFound, "job not found")
		}
		log.L().Error().Err(err).Str("layer", "handler").Str("job_id", jobID).Dur("elapsed", elapsed).Msg("← CMI GetPolicyByJobID failed")
		return dto.Error(c, fiber.StatusInternalServerError, "internal error")
	}

	elapsed := time.Since(start)
	log.L().Info().Str("layer", "handler").Str("job_id", jobID).Dur("elapsed", elapsed).Msg("← CMI GetPolicyByJobID OK")
	return dto.Success(c, fiber.StatusOK, policy)
}
