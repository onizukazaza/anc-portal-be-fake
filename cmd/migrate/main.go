// run cmd 🔑
//
// migrate up (apply all)
// go run ./cmd/migrate/main.go --action up
//
// migrate down (rollback all)
// go run ./cmd/migrate/main.go --action down
//
// migrate steps (step forward/backward)
// go run ./cmd/migrate/main.go --action steps --steps 1
// go run ./cmd/migrate/main.go --action steps --steps -1
//
// show current version
// go run ./cmd/migrate/main.go --action version
//
// force version (fix dirty state)
// go run ./cmd/migrate/main.go --action force --version 1
package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database/postgres"
)

type options struct {
	action        string
	steps         int
	version       int
	migrationPath string
}

func main() {
	// >> Parse CLI flags
	opts := mustParseOptions()

	// >> Load Config
	cfg := mustLoadConfig()

	// >> Build database connection URL
	databaseURL := buildDatabaseURL(cfg)

	// >> Run migration command
	if err := runMigrationCommand(databaseURL, opts.migrationPath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "migration command failed: %v\n", err)
		os.Exit(1)
	}
}

func mustParseOptions() options {
	var opts options

	flag.StringVar(&opts.action, "action", "up", "migration action: up, down, steps, version, force")
	flag.IntVar(&opts.steps, "steps", 0, "number of migration steps, use with --action steps")
	flag.IntVar(&opts.version, "version", 0, "migration version, use with --action force")
	flag.StringVar(&opts.migrationPath, "path", "migrations", "path to migrations directory")
	flag.Parse()

	return opts
}

func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func buildDatabaseURL(cfg *config.Config) string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.Database.User, cfg.Database.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port),
		Path:     cfg.Database.DBName,
		RawQuery: fmt.Sprintf("sslmode=%s", cfg.Database.SSLMode),
	}
	return u.String()
}

func runMigrationCommand(databaseURL, migrationPath string, opts options) error {
	switch opts.action {
	case "up":
		return postgres.MigrateUp(databaseURL, migrationPath)
	case "down":
		return postgres.MigrateDown(databaseURL, migrationPath)
	case "steps":
		if opts.steps == 0 {
			return fmt.Errorf("steps action requires --steps, example: --steps 1 or --steps -1")
		}
		return postgres.MigrateSteps(databaseURL, migrationPath, opts.steps)
	case "version":
		return postgres.ShowMigrationVersion(databaseURL, migrationPath)
	case "force":
		if opts.version < 0 {
			return fmt.Errorf("force action requires --version >= 0")
		}
		return postgres.ForceMigrationVersion(databaseURL, migrationPath, opts.version)
	default:
		return fmt.Errorf("unsupported action: %s", opts.action)
	}
}
