package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/domain"
)

// Repository — data access contract for Example module.
// Implement in adapters/postgres/ (or any other persistence adapter).
type Repository interface {
	FindByID(ctx context.Context, id string) (*domain.Example, error)
	FindAll(ctx context.Context) ([]domain.Example, error)
}
