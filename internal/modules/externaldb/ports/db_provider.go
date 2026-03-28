package ports

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBProvider defines access to external database pools.
type DBProvider interface {
	External(name string) (*pgxpool.Pool, error)
	HealthCheck(ctx context.Context) error
}
