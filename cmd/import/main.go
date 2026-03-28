// run cmd 🔑
//
// Import ข้อมูล user จากไฟล์ CSV เข้า main DB (upsert ทีละ row ใน transaction เดียว)
// go run ./cmd/import/main.go --env .env.local --path .\base_data\users.csv --service_type user
//
// Import ข้อมูล insurer installment (แผนผ่อนชำระ) จากไฟล์ CSV
// go run ./cmd/import/main.go --env .env.local --path .\base_data\insurer_installment.csv --service_type insurer_installment
//
// Import ข้อมูลบริษัทประกัน (insurer) จากไฟล์ CSV
// go run ./cmd/import/main.go --env .env.local --path .\base_data\insurer.csv --service_type insurer
//
// Import ข้อมูลจังหวัด (province) จากไฟล์ CSV
// go run ./cmd/import/main.go --env .env.local --path .\base_data\province.csv --service_type province
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database/postgres"
	importer "github.com/onizukazaza/anc-portal-be-fake/internal/import"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/banner"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

const (
	serviceName = "anc-import-data"
)

type options struct {
	envPath     string
	filePath    string
	serviceType string
}

func main() {
	start := time.Now()

	opts := mustParseOptions()
	cfg := mustLoadConfig(opts.envPath)

	logger := log.New(serviceName)
	log.Set(logger)

	absFilePath, err := filepath.Abs(opts.filePath)
	if err != nil {
		log.L().Fatal().Err(err).Str("file_path", opts.filePath).Msg("failed to resolve csv path")
	}

	if _, err = os.Stat(absFilePath); err != nil {
		log.L().Fatal().Err(err).Str("file_path", absFilePath).Msg("csv file not found")
	}

	log.L().
		Info().
		Str("service_type", opts.serviceType).
		Str("file_path", absFilePath).
		Str("env_path", opts.envPath).
		Msg("starting import data")

	ctx := context.Background()

	dbManager, err := postgres.NewManager(ctx, cfg)
	if err != nil {
		log.L().Fatal().Err(err).Msg("database connection failed")
	}
	defer dbManager.Close()

	log.L().Info().Msg("database connected")

	// >> Print startup banner
	bannerRows := []banner.Row{
		banner.DBRow("Database (main)", cfg.Database.DBName, cfg.Database.Host, cfg.Database.Port),
	}
	bannerRows = append(bannerRows,
		banner.Row{Label: "Service Type", Value: opts.serviceType},
		banner.Row{Label: "File", Value: absFilePath},
	)
	banner.Print(banner.Options{
		AppName:  "ANC Import Data",
		Version:  "1.0.0",
		Env:      cfg.StageStatus,
		BootTime: time.Since(start),
		Rows:     bannerRows,
	})

	if err := importer.Run(importer.RunRequest{
		ServiceType: opts.serviceType,
		FilePath:    absFilePath,
		DB:          dbManager.Main(),
	}); err != nil {
		log.L().Fatal().Err(err).Msg("import failed")
	}

	log.L().
		Info().
		Dur("duration", time.Since(start)).
		Msg("import completed successfully")
}

func mustParseOptions() options {
	var opts options

	flag.StringVar(&opts.envPath, "env", "", "path to env file")
	flag.StringVar(&opts.filePath, "path", "", "path to csv file")
	flag.StringVar(&opts.serviceType, "service_type", "", "import service type เช่น insurer, insurer_installment, province, user")
	flag.Parse()

	if opts.envPath == "" {
		exit("missing flag --env")
	}
	if opts.filePath == "" {
		exit("missing flag --path")
	}
	if opts.serviceType == "" {
		exit("missing flag --service_type")
	}

	return opts
}

func mustLoadConfig(envPath string) *config.Config {
	if _, err := os.Stat(envPath); err != nil {
		exit(fmt.Sprintf("env file not found: %s", envPath))
	}

	if err := godotenv.Overload(envPath); err != nil {
		exit(fmt.Sprintf("failed to load env file: %v", err))
	}

	cfg, err := config.Load()
	if err != nil {
		exit(fmt.Sprintf("failed to load config: %v", err))
	}

	return cfg
}

func exit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
