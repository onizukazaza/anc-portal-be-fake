package postgres

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

const (
	defaultConnectTimeout = 5 * time.Second
	startupPingTimeout    = 5 * time.Second
)

type DB struct {
	pool *pgxpool.Pool
}

// ======================================================
// Generic Constructor (main + external)
// ======================================================
func NewWithConfig(ctx context.Context, dbCfg config.Database, otelEnabled bool) (*DB, error) {
	pgxConfig, err := pgxpool.ParseConfig(buildDSN(dbCfg))
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}

	applyObservability(otelEnabled, pgxConfig)
	applyPoolTuning(dbCfg, pgxConfig)
	applySessionParams(dbCfg, pgxConfig)

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pingWithTimeout(ctx, pool, startupPingTimeout); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &DB{pool: pool}, nil
}

// ======================================================
// Public Methods
// ======================================================
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *DB) Close() {
	if db.pool == nil {
		return
	}
	log.L().Info().Msg("closing database connection pool")
	db.pool.Close()
}

func (db *DB) Health(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// ======================================================
// Internal Helpers
// ======================================================
func buildDSN(dbCfg config.Database) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(dbCfg.User, dbCfg.Password),
		Host:   fmt.Sprintf("%s:%d", dbCfg.Host, dbCfg.Port),
		Path:   dbCfg.DBName,
	}

	q := u.Query()
	q.Set("sslmode", dbCfg.SSLMode)
	u.RawQuery = q.Encode()

	return u.String()
}

// MaskDSN masks the password in a PostgreSQL DSN for safe logging.
// e.g. "postgres://user:secret@host/db" → "postgres://user:****@host/db"
// Handles edge cases: colons in password, percent-encoded chars, no password, etc.
func MaskDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn
	}
	if _, hasPass := u.User.Password(); !hasPass {
		return dsn
	}
	// Replace the encoded userinfo with a readable masked version.
	original := u.User.String() + "@"
	masked := u.User.Username() + ":****@"
	return strings.Replace(dsn, original, masked, 1)
}

func applyObservability(enabled bool, pgxConfig *pgxpool.Config) {
	if enabled {
		pgxConfig.ConnConfig.Tracer = otelpgx.NewTracer()
	}
}

func applyPoolTuning(dbCfg config.Database, pgxConfig *pgxpool.Config) {
	pgxConfig.MaxConns = int32(min(dbCfg.MaxConns, math.MaxInt32)) //nolint:gosec // G115: bounded by min()
	pgxConfig.MinConns = int32(min(dbCfg.MinConns, math.MaxInt32)) //nolint:gosec // G115: bounded by min()
	pgxConfig.MaxConnLifetime = dbCfg.MaxConnLifetime
	pgxConfig.MaxConnIdleTime = dbCfg.MaxConnIdleTime

	ct := dbCfg.ConnectTimeout
	if ct <= 0 {
		ct = defaultConnectTimeout
	}
	pgxConfig.ConnConfig.ConnectTimeout = ct
}

func applySessionParams(dbCfg config.Database, pgxConfig *pgxpool.Config) {
	if pgxConfig.ConnConfig.RuntimeParams == nil {
		pgxConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}

	// statement timeout
	if dbCfg.StatementTimeout > 0 {
		pgxConfig.ConnConfig.RuntimeParams["statement_timeout"] =
			fmt.Sprintf("%dms", dbCfg.StatementTimeout.Milliseconds())
	}

	// application name
	pgxConfig.ConnConfig.RuntimeParams["application_name"] = "anc-portal-be"

	// default schema
	if dbCfg.Schema != "" {
		pgxConfig.ConnConfig.RuntimeParams["search_path"] = dbCfg.Schema
	}
}

func pingWithTimeout(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return pool.Ping(pingCtx)
}
