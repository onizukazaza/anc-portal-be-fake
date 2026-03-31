package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database/mysql"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// >> Manager — orchestrates the main database and all external database connections.
// Lives in the database package (not postgres/) because it manages multiple drivers.
type Manager struct {
	main      *postgres.DB
	externals map[string]ExternalConn
}

// >> NewManager connects to the main database (always postgres) and all configured
// external databases. Each external entry is routed to the correct driver based
// on config.Database.Driver ("postgres" | "mysql"). Default is "postgres".
func NewManager(ctx context.Context, cfg *config.Config) (*Manager, error) {
	// >> Main Database (always postgres)
	mainDB, err := postgres.NewWithConfig(ctx, cfg.Database, cfg.OTel.Enabled)
	if err != nil {
		return nil, fmt.Errorf("main database connection failed: %w", err)
	}
	log.L().Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("db", cfg.Database.DBName).
		Msg("main database connected")

	// >> External Databases (multi-driver: postgres, mysql)
	externals := make(map[string]ExternalConn)
	for name, dbCfg := range cfg.ExternalDBs { //nolint:gocritic // rangeValCopy: read-only config, copy is acceptable
		conn, err := connectExternal(ctx, name, &dbCfg, cfg.OTel.Enabled)
		if err != nil {
			// close already-opened connections before returning
			for _, c := range externals {
				c.Close()
			}
			return nil, fmt.Errorf("external database [%s] connection failed: %w", name, err)
		}
		externals[name] = conn
		log.L().Info().
			Str("name", name).
			Str("driver", conn.Driver()).
			Str("host", dbCfg.Host).
			Int("port", dbCfg.Port).
			Str("db", dbCfg.DBName).
			Msg("external database connected")
	}

	if len(externals) == 0 {
		log.L().Warn().Msg("no external databases configured")
	}

	return &Manager{main: mainDB, externals: externals}, nil
}

// >> connectExternal creates the appropriate driver connection based on config.Driver.
// Default driver is "postgres" when Driver is empty.
func connectExternal(ctx context.Context, name string, dbCfg *config.Database, otelEnabled bool) (ExternalConn, error) {
	driver := dbCfg.Driver
	if driver == "" {
		driver = DriverPostgres
	}

	switch driver {
	case DriverPostgres:
		return postgres.NewWithConfig(ctx, *dbCfg, otelEnabled)

	case DriverMySQL:
		return mysql.NewWithConfig(ctx, *dbCfg)

	default:
		return nil, fmt.Errorf("unsupported database driver: %q (name=%s)", driver, name)
	}
}

// >> Main returns main DB pool (always postgres).
func (m *Manager) Main() *pgxpool.Pool {
	return m.main.Pool()
}

// >> External returns external database connection by name.
func (m *Manager) External(name string) (ExternalConn, error) {
	conn, ok := m.externals[name]
	if !ok {
		return nil, fmt.Errorf("external database not found: %s", name)
	}
	return conn, nil
}

// >> Read returns the main DB pool for read operations.
func (m *Manager) Read() *pgxpool.Pool {
	return m.main.Pool()
}

// >> Write returns the main DB pool for write operations.
func (m *Manager) Write() *pgxpool.Pool {
	return m.main.Pool()
}

// >> HealthCheck verifies connectivity for main and external databases.
func (m *Manager) HealthCheck(ctx context.Context) error {
	if err := m.main.Health(ctx); err != nil {
		return fmt.Errorf("main database unhealthy: %w", err)
	}

	for name, conn := range m.externals {
		if err := conn.Health(ctx); err != nil {
			return fmt.Errorf("external database [%s] unhealthy: %w", name, err)
		}
	}

	return nil
}

// >> Close releases all database connections.
func (m *Manager) Close() {
	if m.main != nil {
		m.main.Close()
	}
	for _, conn := range m.externals {
		conn.Close()
	}
}
