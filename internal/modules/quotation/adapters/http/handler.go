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
// @Failure 400 {object} dto.ErrorResponse "[11001] qt-id-required — ไม่ได้ส่ง quotation id"
// @Failure 404 {object} dto.ErrorResponse "[11002] qt-not-found — ไม่พบ quotation"
// @Failure 500 {object} dto.ErrorResponse "[11003] qt-internal-error — เกิดข้อผิดพลาดภายใน quotation service"
// @Security BearerAuth
// @Router /quotations/{id} [get]
func (h *Handler) GetByID(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationHandler).Start(c.UserContext(), "GetByID")
	defer span.End()

	id := c.Params("id")
	if id == "" {
		return dto.ErrorWithTrace(c, fiber.StatusBadRequest, "quotation id is required", dto.TraceQTIdRequired)
	}

	qt, err := h.service.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			return dto.ErrorWithTrace(c, fiber.StatusNotFound, "quotation not found", dto.TraceQTNotFound)
		}
		span.RecordError(err)
		return dto.ErrorWithTrace(c, fiber.StatusInternalServerError, "internal error", dto.TraceQTInternalError)
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
// @Failure 400 {object} dto.ErrorResponse "[11004] qt-customer-id-required — ไม่ได้ส่ง customerId"
// @Failure 500 {object} dto.ErrorResponse "[11005] qt-list-internal-error — เกิดข้อผิดพลาดขณะดึงรายการ quotation"
// @Security BearerAuth
// @Router /quotations [get]
func (h *Handler) ListByCustomer(c *fiber.Ctx) error {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationHandler).Start(c.UserContext(), "ListByCustomer")
	defer span.End()

	customerID := c.Query("customerId")
	if customerID == "" {
		return dto.ErrorWithTrace(c, fiber.StatusBadRequest, "customerId is required", dto.TraceQTCustomerRequired)
	}

	pg := pagination.FromFiber(c)

	result, err := h.service.ListByCustomer(ctx, customerID, pg)
	if err != nil {
		span.RecordError(err)
		return dto.ErrorWithTrace(c, fiber.StatusInternalServerError, "internal error", dto.TraceQTListInternalError)
	}

	return dto.Success(c, fiber.StatusOK, result)
}
