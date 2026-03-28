package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/validator"
)

type Handler struct {
	service *app.Service
}

type loginRequest struct {
	Username string `json:"username" validate:"required" example:"admin"`
	Password string `json:"password" validate:"required" example:"secret123"`
}

func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

func NewAuthController(service *app.Service) AuthController {
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
// @Failure 400 {object} dto.ApiResponse "Invalid request body"
// @Failure 401 {object} dto.ApiResponse "Invalid credentials"
// @Failure 500 {object} dto.ApiResponse "Internal error"
// @Router /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerAuthHandler).Start(c.UserContext(), "Login")
	defer span.End()

	var req loginRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return nil // response already sent
	}

	session, err := h.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, app.ErrInvalidCredentials) {
			return dto.Error(c, fiber.StatusUnauthorized, "invalid credentials")
		}
		return dto.Error(c, fiber.StatusInternalServerError, "internal error")
	}

	return dto.Success(c, fiber.StatusOK, session)
}
