package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// >> Supported driver identifiers
const (
	DriverPostgres = "postgres"
	DriverMySQL    = "mysql"
)

// >> ExternalConn — driver-agnostic interface for external database connections.
// Every driver (postgres, mysql, etc.) implements this interface.
type ExternalConn interface {
	Health(ctx context.Context) error                                   // >> connectivity check
	Close()                                                             // >> release connection pool
	Driver() string                                                     // >> driver name ("postgres", "mysql")
	Diagnostic(ctx context.Context) (dbName, version string, err error) // >> DB name + server version
}

// >> Type-safe helpers — extract the concrete connection from ExternalConn.
// Modules call these instead of doing raw type assertions.

type pgxPooler interface{ Pool() *pgxpool.Pool } // >> satisfied by postgres
type sqlDBer interface{ DB() *sql.DB }           // >> satisfied by mysql

// >> PgxPool extracts *pgxpool.Pool from a postgres ExternalConn.
func PgxPool(conn ExternalConn) (*pgxpool.Pool, error) {
	if conn == nil {
		return nil, fmt.Errorf("database: ExternalConn is nil")
	}
	if p, ok := conn.(pgxPooler); ok {
		return p.Pool(), nil
	}
	return nil, fmt.Errorf("database: driver %q does not provide *pgxpool.Pool", conn.Driver())
}

// >> SQLDB extracts *sql.DB from a mysql (or sql.DB-based) ExternalConn.
func SQLDB(conn ExternalConn) (*sql.DB, error) {
	if conn == nil {
		return nil, fmt.Errorf("database: ExternalConn is nil")
	}
	if s, ok := conn.(sqlDBer); ok {
		return s.DB(), nil
	}
	return nil, fmt.Errorf("database: driver %q does not provide *sql.DB", conn.Driver())
}
