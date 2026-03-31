// run cmd 🔑
//
// Full sync ตาราง quotations (ลบข้อมูลเก่า + insert ใหม่ทั้งหมดจาก external DB)
// go run ./cmd/sync/main.go --env .env.local --table quotations --mode full
//
// Incremental sync ตาราง quotations (sync เฉพาะ row ที่เปลี่ยนแปลงใน 24 ชม. ที่ผ่านมา)
// go run ./cmd/sync/main.go --env .env.local --table quotations --mode incremental --since 24h
//
// Full sync ทุกตาราง พร้อมกำหนด batch size = 1000 rows ต่อ batch
// go run ./cmd/sync/main.go --env .env.local --table all --mode full --batch-size 1000
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database"
	datasync "github.com/onizukazaza/anc-portal-be-fake/internal/sync"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/banner"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

const serviceName = "anc-sync-data"

type options struct {
	envPath   string
	table     string
	mode      string
	batchSize int
	since     string
}

func main() {
	start := time.Now()

	opts := mustParseOptions()
	cfg := mustLoadConfig(opts.envPath)

	logger := log.New(serviceName)
	log.Set(logger)

	log.L().Info().
		Str("table", opts.table).
		Str("mode", opts.mode).
		Int("batch_size", opts.batchSize).
		Msg("starting sync")

	// graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// >> Connect database manager (main + external)
	dbManager, err := database.NewManager(ctx, cfg)
	if err != nil {
		log.L().Fatal().Err(err).Msg("database connection failed")
	}
	defer dbManager.Close()

	log.L().Info().Msg("database connected")

	// >> Print startup banner
	bannerRows := []banner.Row{
		banner.DBRow("Database (main)", cfg.Database.DBName, cfg.Database.Host, cfg.Database.Port),
	}
	for name := range cfg.ExternalDBs {
		dbCfg := cfg.ExternalDBs[name]
		bannerRows = append(bannerRows, banner.DBRow("Database ("+name+")", dbCfg.DBName, dbCfg.Host, dbCfg.Port))
	}
	bannerRows = append(bannerRows,
		banner.Row{Label: "Table", Value: opts.table},
		banner.Row{Label: "Mode", Value: opts.mode},
		banner.Row{Label: "Batch Size", Value: fmt.Sprintf("%d", opts.batchSize)},
	)
	banner.Print(banner.Options{
		AppName:  "ANC Sync Data",
		Version:  "1.0.0",
		Env:      cfg.StageStatus,
		BootTime: time.Since(start),
		Rows:     bannerRows,
	})

	// >> Build registry — เพิ่ม syncer ใหม่ตรงนี้
	registry := datasync.NewRegistry()
	registerSyncers(registry, dbManager, cfg)

	runner := datasync.NewRunner(registry)

	// >> Build sync request
	req := buildSyncRequest(opts)

	// >> Run
	if opts.table == "all" {
		results, err := runner.RunAll(ctx, req)
		if err != nil {
			log.L().Fatal().Err(err).Msg("sync all failed")
		}
		for _, r := range results {
			printResult(r)
		}
	} else {
		result, err := runner.RunOne(ctx, opts.table, req)
		if err != nil {
			log.L().Fatal().Err(err).Msg("sync failed")
		}
		printResult(result)
	}

	log.L().Info().
		Dur("total_duration", time.Since(start)).
		Msg("sync completed successfully")
}

// registerSyncers ลงทะเบียน syncer ทั้งหมด.
// เพิ่มตารางใหม่: สร้างไฟล์ใน internal/sync/ + register ที่นี่.
func registerSyncers(registry *datasync.Registry, db *database.Manager, cfg *config.Config) {
	// quotations: source = meprakun (external), dest = main DB
	if conn, err := db.External("meprakun"); err == nil {
		pool, pgErr := database.PgxPool(conn)
		if pgErr != nil {
			log.L().Warn().Err(pgErr).Msg("skip quotation syncer: meprakun is not postgres")
		} else {
			registry.Register(datasync.NewQuotationSyncer(pool, db.Main()))
			log.L().Info().Msg("registered syncer: quotations (meprakun → main)")
		}
	} else {
		log.L().Warn().Err(err).Msg("skip quotation syncer: external DB 'meprakun' not available")
	}

	// เพิ่ม syncer ใหม่ได้ตรงนี้:
	// registry.Register(datasync.NewCustomerSyncer(pool, db.Main()))
	// registry.Register(datasync.NewPolicySyncer(pool, db.Main()))
	_ = cfg // ใช้ config สำหรับ syncer ที่ต้องการ setting เพิ่ม
}

func buildSyncRequest(opts options) datasync.SyncRequest {
	req := datasync.SyncRequest{
		Mode:      datasync.Mode(opts.mode),
		BatchSize: opts.batchSize,
	}

	if opts.since != "" {
		d, err := time.ParseDuration(opts.since)
		if err != nil {
			log.L().Fatal().Err(err).Str("since", opts.since).Msg("invalid --since duration")
		}
		req.Since = time.Now().Add(-d)
	}

	return req
}

func printResult(r *datasync.SyncResult) {
	log.L().Info().
		Str("table", r.Table).
		Str("mode", string(r.Mode)).
		Int("total", r.Total).
		Int("inserted", r.Inserted).
		Int("updated", r.Updated).
		Int("skipped", r.Skipped).
		Int("errors", r.Errors).
		Dur("duration", r.Duration).
		Msg("sync result")
}

func mustParseOptions() options {
	var opts options

	flag.StringVar(&opts.envPath, "env", "", "path to env file")
	flag.StringVar(&opts.table, "table", "", "table name to sync (or 'all')")
	flag.StringVar(&opts.mode, "mode", "full", "sync mode: full | incremental")
	flag.IntVar(&opts.batchSize, "batch-size", datasync.DefaultBatchSize, "rows per batch")
	flag.StringVar(&opts.since, "since", "", "incremental: duration lookback (e.g. 24h, 1h30m)")
	flag.Parse()

	if opts.envPath == "" {
		exit("missing flag --env")
	}
	if opts.table == "" {
		exit("missing flag --table")
	}

	opts.mode = strings.ToLower(opts.mode)
	if opts.mode != "full" && opts.mode != "incremental" {
		exit("--mode must be 'full' or 'incremental'")
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
