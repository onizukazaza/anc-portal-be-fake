package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/ports"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

type Handler struct {
	service ports.CMIPolicyService
}

func NewHandler(service ports.CMIPolicyService) *Handler {
	return &Handler{service: service}
}

func NewCMIController(service ports.CMIPolicyService) ports.CMIController {
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
// @Failure      400 {object} dto.ErrorResponse "[12001] cmi-job-id-required — ไม่ได้ส่ง job_id"
// @Failure      404 {object} dto.ErrorResponse "[12002] cmi-job-not-found — ไม่พบ job"
// @Failure      500 {object} dto.ErrorResponse "[12003] cmi-internal-error — เกิดข้อผิดพลาดภายใน CMI service"
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
		return dto.ErrorWithTrace(c, fiber.StatusBadRequest, "job_id is required", dto.TraceCMIJobIdRequired)
	}

	policy, err := h.service.GetPolicyByJobID(ctx, jobID)
	if err != nil {
		elapsed := time.Since(start)
		if errors.Is(err, app.ErrJobNotFound) {
			log.L().Warn().Str("layer", "handler").Str("job_id", jobID).Dur("elapsed", elapsed).Msg("← CMI job not found")
			return dto.ErrorWithTrace(c, fiber.StatusNotFound, "job not found", dto.TraceCMIJobNotFound)
		}
		log.L().Error().Err(err).Str("layer", "handler").Str("job_id", jobID).Dur("elapsed", elapsed).Msg("← CMI GetPolicyByJobID failed")
		return dto.ErrorWithTrace(c, fiber.StatusInternalServerError, "internal error", dto.TraceCMIInternalError)
	}

	elapsed := time.Since(start)
	log.L().Info().Str("layer", "handler").Str("job_id", jobID).Dur("elapsed", elapsed).Msg("← CMI GetPolicyByJobID OK")
	return dto.Success(c, fiber.StatusOK, policy)
}
