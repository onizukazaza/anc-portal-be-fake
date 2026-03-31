package ports

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/database"
)

// DBProvider defines access to external database connections.
type DBProvider interface {
	External(name string) (database.ExternalConn, error)
	HealthCheck(ctx context.Context) error
}
