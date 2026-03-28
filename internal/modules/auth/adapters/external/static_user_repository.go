package external

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
)

// StaticUserRepository is a dev-friendly adapter used until postgres adapter is implemented.
type StaticUserRepository struct {
	users map[string]domain.User
}

func NewStaticUserRepository() *StaticUserRepository {
	return &StaticUserRepository{
		users: map[string]domain.User{
			"admin": {
				ID:           "u-001",
				Username:     "admin",
				PasswordHash: "admin123",
				Roles:        []string{"admin"},
			},
		},
	}
}

func (r *StaticUserRepository) FindByUsername(_ context.Context, username string) (*domain.User, error) {
	user, ok := r.users[username]
	if !ok {
		return nil, nil
	}
	return &user, nil
}
