package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/ports"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/validator"
)

type Handler struct {
	service ports.AuthService
}

type loginRequest struct {
	Username string `json:"username" validate:"required" example:"admin"`
	Password string `json:"password" validate:"required" example:"secret123"`
}

func NewHandler(service ports.AuthService) *Handler {
	return &Handler{service: service}
}

func NewAuthController(service ports.AuthService) ports.AuthController {
	return &Handler{service: service}
}

// Login godoc
// @Summary Login user
// @Description Authenticate user with username and password, then return access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body loginRequest true "Login payload"
// @Success 200 {object} dto.ApiResponse "Login success"
// @Failure 400 {object} dto.ErrorResponse "[10001] auth-bind-failed — request body ไม่ถูกต้อง"
// @Failure 401 {object} dto.ErrorResponse "[10002] auth-invalid-creds — username/password ไม่ถูกต้อง"
// @Failure 500 {object} dto.ErrorResponse "[10003] auth-internal-error — เกิดข้อผิดพลาดภายใน auth service"
// @Router /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerAuthHandler).Start(c.UserContext(), "Login")
	defer span.End()

	var req loginRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return err
	}

	session, err := h.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, app.ErrInvalidCredentials) {
			return dto.ErrorWithTrace(c, fiber.StatusUnauthorized, "invalid credentials", dto.TraceAuthBadLogin)
		}
		return dto.ErrorWithTrace(c, fiber.StatusInternalServerError, "internal error", dto.TraceAuthInternalError)
	}

	return dto.Success(c, fiber.StatusOK, session)
}
