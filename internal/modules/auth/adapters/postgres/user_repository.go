package postgres

import (
	"context"
	"errors"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerAuthRepo).Start(ctx, "FindByUsername")
	defer span.End()

	const q = `
		SELECT id, username, password_hash, roles
		FROM users
		WHERE username = $1
		LIMIT 1
	`

	var user domain.User
	if err := r.pool.QueryRow(ctx, q, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Roles); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}
