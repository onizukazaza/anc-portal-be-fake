package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
)

// Handler implements WebhookController.
type Handler struct {
	service *app.Service
}

// NewHandler creates a new webhook handler.
func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

// NewWebhookController creates the controller interface.
func NewWebhookController(service *app.Service) WebhookController {
	return &Handler{service: service}
}

// HandleGitHubPush godoc
// @Summary Receive GitHub push webhook
// @Description Receives a push event from GitHub webhook and sends notification to Discord
// @Tags Webhook
// @Accept json
// @Produce json
// @Param X-Hub-Signature-256 header string false "GitHub HMAC signature"
// @Param X-GitHub-Event header string true "GitHub event type"
// @Success 200 {object} dto.ApiResponse "Event processed"
// @Failure 400 {object} dto.ApiResponse "Unsupported event"
// @Failure 401 {object} dto.ErrorResponse "[14001] wh-invalid-signature — GitHub signature ไม่ถูกต้อง"
// @Failure 500 {object} dto.ErrorResponse "[14002] wh-process-failed — ประมวลผล webhook ล้มเหลว"
// @Router /webhooks/github [post]
func (h *Handler) HandleGitHubPush(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerWebhookHandler).Start(c.UserContext(), "HandleGitHubPush")
	defer span.End()

	eventType := c.Get("X-GitHub-Event")
	if eventType != "push" {
		return dto.Success(c, fiber.StatusOK, fiber.Map{"message": "event ignored", "event": eventType})
	}

	signature := c.Get("X-Hub-Signature-256")
	rawBody := c.Body()

	if err := h.service.HandlePush(ctx, rawBody, signature); err != nil {
		if errors.Is(err, app.ErrInvalidSignature) {
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "invalid signature", dto.TraceWHInvalidSignature)
		}
		return dto.ErrorWithTrace(c, fiber.StatusInternalServerError, "failed to process webhook", dto.TraceWHProcessFailed)
	}

	return dto.Success(c, fiber.StatusOK, fiber.Map{"message": "push event processed"})
}
