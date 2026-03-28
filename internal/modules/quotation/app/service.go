package app

import (
	"context"
	"errors"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/ports"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/pagination"
)

var ErrNotFound = errors.New("quotation not found")

type Service struct {
	repo ports.QuotationRepository
}

func NewService(repo ports.QuotationRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.Quotation, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationService).Start(ctx, "GetByID")
	defer span.End()

	qt, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if qt == nil {
		return nil, ErrNotFound
	}
	return qt, nil
}

func (s *Service) ListByCustomer(ctx context.Context, customerID string, pg pagination.Request) (pagination.Response[domain.Quotation], error) {
	ctx, span := appOtel.Tracer(appOtel.TracerQuotationService).Start(ctx, "ListByCustomer")
	defer span.End()

	items, total, err := s.repo.FindByCustomerID(ctx, customerID, pg)
	if err != nil {
		return pagination.Response[domain.Quotation]{}, err
	}
	return pagination.NewResponse(items, total, pg), nil
}
