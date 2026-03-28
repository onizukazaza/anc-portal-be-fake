package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
)

type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
}
