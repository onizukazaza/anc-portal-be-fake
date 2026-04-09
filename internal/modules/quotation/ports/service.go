package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/pagination"
)

// QuotationService handles quotation business logic.
type QuotationService interface {
	// GetByID retrieves a single quotation by ID.
	GetByID(ctx context.Context, id string) (*domain.Quotation, error)

	// ListByCustomer retrieves paginated quotations for a customer.
	ListByCustomer(ctx context.Context, customerID string, pg pagination.Request) (pagination.Response[domain.Quotation], error)
}
