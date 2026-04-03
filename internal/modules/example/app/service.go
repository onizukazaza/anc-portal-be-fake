package app

import (
	"context"
	"errors"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/ports"
)

var ErrNotFound = errors.New("example not found")

// Service — business logic for Example module.
// Depends on ports (interface), not on concrete adapters.
type Service struct {
	repo ports.Repository
}

func NewService(repo ports.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.Example, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNotFound
	}
	return item, nil
}

func (s *Service) List(ctx context.Context) ([]domain.Example, error) {
	return s.repo.FindAll(ctx)
}
