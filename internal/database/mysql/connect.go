package mysql

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

const (
	defaultConnectTimeout = 5 * time.Second
	startupPingTimeout    = 5 * time.Second
)

// >> DB wraps *sql.DB with MySQL-specific configuration.
// Implements database.ExternalConn.
type DB struct {
	db *sql.DB
}

// >> Constructor
func NewWithConfig(ctx context.Context, dbCfg config.Database) (*DB, error) {
	dsnCfg, err := buildMySQLConfig(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("build mysql config: %w", err)
	}

	db, err := sql.Open("mysql", dsnCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	applyPoolTuning(dbCfg, db)

	if err := pingWithTimeout(ctx, db, startupPingTimeout); err != nil {
		db.Close()
		return nil, fmt.Errorf("mysql ping failed: %w", err)
	}

	return &DB{db: db}, nil
}

// >> Public Methods (implements database.ExternalConn)

// >> DB returns the underlying *sql.DB for use by repositories.
func (d *DB) DB() *sql.DB { return d.db }

// >> Close releases all connections in the pool.
func (d *DB) Close() {
	if d.db == nil {
		return
	}
	log.L().Info().Msg("closing mysql connection pool")
	d.db.Close()
}

// >> Health checks connectivity.
func (d *DB) Health(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// >> Driver returns "mysql".
func (d *DB) Driver() string { return "mysql" }

// >> Diagnostic returns the current database name and MySQL server version.
func (d *DB) Diagnostic(ctx context.Context) (string, string, error) {
	var dbName, version string
	if err := d.db.QueryRowContext(ctx, "SELECT DATABASE(), VERSION()").Scan(&dbName, &version); err != nil {
		return "", "", err
	}
	return dbName, version, nil
}

// >> Internal Helpers

func buildMySQLConfig(dbCfg config.Database) (*mysql.Config, error) {
	cfg := mysql.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", dbCfg.Host, dbCfg.Port)
	cfg.User = dbCfg.User
	cfg.Passwd = dbCfg.Password
	cfg.DBName = dbCfg.DBName
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.MultiStatements = false // ป้องกัน SQL injection via stacked queries

	// connection timeout
	ct := dbCfg.ConnectTimeout
	if ct <= 0 {
		ct = defaultConnectTimeout
	}
	cfg.Timeout = ct

	// statement timeout → max_execution_time (MySQL 5.7.8+)
	if dbCfg.StatementTimeout > 0 {
		if cfg.Params == nil {
			cfg.Params = make(map[string]string)
		}
		cfg.Params["max_execution_time"] = fmt.Sprintf("%d", dbCfg.StatementTimeout.Milliseconds())
	}

	// TLS/SSL — map SSLMode config to MySQL TLS config
	if err := applyTLS(dbCfg.SSLMode, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// >> applyTLS maps SSLMode to MySQL TLS configuration.
//
//	"disable" → cleartext | "require" → TLS (skip verify) | "verify-ca"/"verify" → TLS (verify cert)
func applyTLS(sslMode string, cfg *mysql.Config) error {
	switch sslMode {
	case "disable", "":
		cfg.TLSConfig = "" // cleartext
	case "require":
		// TLS on, but don't verify server identity (like Postgres sslmode=require)
		tlsCfg := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec // intentional: "require" mode skips cert verification
		}
		tlsKey := "custom-require"
		if err := mysql.RegisterTLSConfig(tlsKey, tlsCfg); err != nil {
			return fmt.Errorf("register TLS config: %w", err)
		}
		cfg.TLSConfig = tlsKey
	case "verify-ca", "verify":
		tlsCfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		tlsKey := "custom-verify"
		if err := mysql.RegisterTLSConfig(tlsKey, tlsCfg); err != nil {
			return fmt.Errorf("register TLS config: %w", err)
		}
		cfg.TLSConfig = tlsKey
	default:
		return fmt.Errorf("unsupported mysql sslMode: %q", sslMode)
	}
	return nil
}

func applyPoolTuning(dbCfg config.Database, db *sql.DB) {
	db.SetMaxOpenConns(min(dbCfg.MaxConns, math.MaxInt32))
	db.SetMaxIdleConns(min(dbCfg.MinConns, math.MaxInt32))
	db.SetConnMaxLifetime(dbCfg.MaxConnLifetime)
	db.SetConnMaxIdleTime(dbCfg.MaxConnIdleTime)
}

func pingWithTimeout(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return db.PingContext(pingCtx)
}
