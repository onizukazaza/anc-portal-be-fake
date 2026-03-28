package http

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/pagination"
)

type Handler struct {
	service *app.Service
}

func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

func NewQuotationController(service *app.Service) QuotationController {
	return &Handler{service: service}
}

// GetByID godoc
// @Summary Get quotation by ID
// @Description Retrieve a single quotation from external ERP database
// @Tags Quotation
// @Accept json
// @Produce json
// @Param id path string true "Quotation ID"
// @Success 200 {object} dto.ApiResponse "Quotation data"
// @Failure 404 {object} dto.ApiResponse "Quotation not found"
// @Failure 500 {object} dto.ApiResponse "Internal error"
// @Security BearerAuth
// @Router /quotations/{id} [get]
func (h *Handler) GetByID(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationHandler).Start(c.UserContext(), "GetByID")
	defer span.End()

	id := c.Params("id")
	if id == "" {
		return dto.Error(c, fiber.StatusBadRequest, "quotation id is required")
	}

	qt, err := h.service.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			return dto.Error(c, fiber.StatusNotFound, "quotation not found")
		}
		return dto.Error(c, fiber.StatusInternalServerError, "internal error")
	}

	return dto.Success(c, fiber.StatusOK, qt)
}

// ListByCustomer godoc
// @Summary List quotations by customer
// @Description Retrieve paginated quotations for a given customer from external ERP database
// @Tags Quotation
// @Accept json
// @Produce json
// @Param customerId query string true "Customer ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort query string false "Sort column" Enums(created_at, doc_no, total_amount, status)
// @Param order query string false "Sort order" Enums(asc, desc) default(desc)
// @Success 200 {object} dto.ApiResponse "Paginated quotations"
// @Failure 400 {object} dto.ApiResponse "Customer ID is required"
// @Failure 500 {object} dto.ApiResponse "Internal error"
// @Security BearerAuth
// @Router /quotations [get]
func (h *Handler) ListByCustomer(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationHandler).Start(c.UserContext(), "ListByCustomer")
	defer span.End()

	customerID := c.Query("customerId")
	if customerID == "" {
		return dto.Error(c, fiber.StatusBadRequest, "customerId is required")
	}

	pg := pagination.FromFiber(c)

	result, err := h.service.ListByCustomer(ctx, customerID, pg)
	if err != nil {
		return dto.Error(c, fiber.StatusInternalServerError, "internal error")
	}

	return dto.Success(c, fiber.StatusOK, result)
}
