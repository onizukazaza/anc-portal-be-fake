package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
)

// AuthService handles authentication business logic.
type AuthService interface {
	// Login authenticates a user and returns a session with access token.
	Login(ctx context.Context, username string, password string) (*domain.Session, error)
}
