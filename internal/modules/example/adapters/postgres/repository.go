package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/domain"
)

// Repository — PostgreSQL implementation of ports.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) FindByID(ctx context.Context, id string) (*domain.Example, error) {
	query := `SELECT id, name, created_at FROM examples WHERE id = $1`

	var item domain.Example
	err := r.pool.QueryRow(ctx, query, id).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) FindAll(ctx context.Context) ([]domain.Example, error) {
	query := `SELECT id, name, created_at FROM examples ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Example
	for rows.Next() {
		var item domain.Example
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
