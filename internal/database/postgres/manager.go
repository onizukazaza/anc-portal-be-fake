package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

type Manager struct {
	main      *DB
	externals map[string]*DB
}

// ------------------------------
// Constructor
// ------------------------------
func NewManager(ctx context.Context, cfg *config.Config) (*Manager, error) {
	// >> Main Database
	mainDB, err := NewWithConfig(ctx, cfg.Database, cfg.OTel.Enabled)
	if err != nil {
		return nil, fmt.Errorf("main database connection failed: %w", err)
	}
	log.L().Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("db", cfg.Database.DBName).
		Msg("main database connected")

	// >> External Databases
	externals := make(map[string]*DB)
	for name := range cfg.ExternalDBs {
		dbCfg := cfg.ExternalDBs[name]
		extDB, err := NewWithConfig(ctx, dbCfg, cfg.OTel.Enabled)
		if err != nil {
			return nil, fmt.Errorf("external database [%s] connection failed: %w", name, err)
		}
		externals[name] = extDB
		log.L().Info().
			Str("name", name).
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

// Main returns main DB pool.
func (m *Manager) Main() *pgxpool.Pool {
	return m.main.Pool()
}

// External returns external DB pool by name.
func (m *Manager) External(name string) (*pgxpool.Pool, error) {
	db, ok := m.externals[name]
	if !ok {
		return nil, fmt.Errorf("external database not found: %s", name)
	}
	return db.Pool(), nil
}

func (m *Manager) Read() *pgxpool.Pool {
	return m.main.Pool()
}

func (m *Manager) Write() *pgxpool.Pool {
	return m.main.Pool()
}

// HealthCheck verifies connectivity for main and external databases.
func (m *Manager) HealthCheck(ctx context.Context) error {
	if err := m.main.Health(ctx); err != nil {
		return fmt.Errorf("main database unhealthy: %w", err)
	}

	for name, db := range m.externals {
		if err := db.Health(ctx); err != nil {
			return fmt.Errorf("external database [%s] unhealthy: %w", name, err)
		}
	}

	return nil
}

// Close releases all database connections.
func (m *Manager) Close() {
	if m.main != nil {
		m.main.Close()
	}
	for _, db := range m.externals {
		db.Close()
	}
}
