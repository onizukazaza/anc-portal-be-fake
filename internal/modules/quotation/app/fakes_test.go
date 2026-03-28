package app

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/pagination"
)

// ─── Fake QuotationRepository ───

type fakeQuotationRepo struct {
	quotation *domain.Quotation
	list      []domain.Quotation
	total     int
	err       error
}

func (f *fakeQuotationRepo) FindByID(_ context.Context, _ string) (*domain.Quotation, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.quotation, nil
}

func (f *fakeQuotationRepo) FindByCustomerID(_ context.Context, _ string, _ pagination.Request) ([]domain.Quotation, int, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.list, f.total, nil
}
