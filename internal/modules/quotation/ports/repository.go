package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/pagination"
)

// QuotationRepository defines read access to quotation data from external database.
type QuotationRepository interface {
	FindByID(ctx context.Context, id string) (*domain.Quotation, error)
	FindByCustomerID(ctx context.Context, customerID string, pg pagination.Request) ([]domain.Quotation, int, error)
}
