package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// >> Provider — database access contract for application modules.
// Modules depend on this interface, never on concrete Manager.
type Provider interface {
	// >> Main pool (always postgres)
	Main() *pgxpool.Pool

	// >> External database connection by name.
	// Returns ExternalConn that may wrap *pgxpool.Pool (postgres) or *sql.DB (mysql).
	// Use database.PgxPool() or database.SQLDB() helpers for type-safe access.
	External(name string) (ExternalConn, error)

	// >> Read / Write pools (currently same as Main)
	Read() *pgxpool.Pool
	Write() *pgxpool.Pool

	// >> Health check and cleanup
	HealthCheck(ctx context.Context) error
	Close()
}
