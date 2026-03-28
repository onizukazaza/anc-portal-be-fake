// run cmd 🔑
//
// Seed ข้อมูล user เริ่มต้น (admin, ops, viewer) เข้า main DB พร้อม hash password ด้วย bcrypt
// go run ./cmd/seed/main.go --env .env.local --service_type auth_user
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database/seed"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/banner"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

const (
	serviceName = "anc-seed-data"
)

type options struct {
	envPath     string
	serviceType string
}

func main() {
	bootStart := time.Now()

	// >> Parse CLI flags
	opts := mustParseOptions()

	// >> Load environment variables from file
	mustLoadEnv(opts.envPath)

	// >> Load Config
	cfg := mustLoadConfig()

	// >> Logger setup
	logger := log.New(serviceName)
	log.Set(logger)

	ctx := context.Background()

	// >> Connect database manager (main + external)
	dbManager, err := postgres.NewManager(ctx, cfg)
	if err != nil {
		log.L().Fatal().Err(err).Msg("database connection failed")
	}
	defer dbManager.Close()

	log.L().Info().Str("service_type", opts.serviceType).Msg("database connected")

	// >> Print startup banner
	bannerRows := []banner.Row{
		banner.DBRow("Database (main)", cfg.Database.DBName, cfg.Database.Host, cfg.Database.Port),
	}
	for name := range cfg.ExternalDBs {
		dbCfg := cfg.ExternalDBs[name]
		bannerRows = append(bannerRows, banner.DBRow("Database ("+name+")", dbCfg.DBName, dbCfg.Host, dbCfg.Port))
	}
	bannerRows = append(bannerRows, banner.Row{Label: "Service Type", Value: opts.serviceType})
	banner.Print(banner.Options{
		AppName:  "ANC Seed Data",
		Version:  "1.0.0",
		Env:      cfg.StageStatus,
		BootTime: time.Since(bootStart),
		Rows:     bannerRows,
	})

	// >> Run seed by service type
	if err := seed.Run(ctx, dbManager.Main(), opts.serviceType); err != nil {
		log.L().Fatal().Err(err).Msg("seed failed")
	}

	log.L().Info().Str("service_type", opts.serviceType).Msg("seed completed successfully")
}

func mustParseOptions() options {
	var opts options

	flag.StringVar(&opts.envPath, "env", "", "path to env file")
	flag.StringVar(&opts.serviceType, "service_type", "", "seed service type เช่น auth_user")
	flag.Parse()

	if opts.envPath == "" {
		fmt.Fprintln(os.Stderr, "missing flag --env")
		os.Exit(1)
	}

	if opts.serviceType == "" {
		fmt.Fprintln(os.Stderr, "missing flag --service_type")
		os.Exit(1)
	}

	return opts
}

func mustLoadEnv(envPath string) {
	if err := godotenv.Overload(envPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load env: %v\n", err)
		os.Exit(1)
	}
}

func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	return cfg
}
