// run cmd 🔑
//
// เริ่ม API server (Fiber) ที่ port 20000 พร้อม middleware, routes, health check
// go run ./cmd/api/main.go
//
// เริ่มแบบ hot-reload ด้วย air (auto restart เมื่อไฟล์เปลี่ยน)
// air -c .air.local.toml

// @title ANC Portal API
// @version 1.0.0
// @description Backend API for ANC Insurance Portal
// @host localhost:20000
// @BasePath /v1
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Input: Bearer {access_token}
//
// NOTE:
// ค่าข้างบนใช้สำหรับ generate Swagger documentation (swag init) เท่านั้น
// ตอน runtime จะถูก override ด้วย environment configuration
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/docs" // >> swagger generated docs
	"github.com/onizukazaza/anc-portal-be-fake/internal/database/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/banner"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/cache"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/kafka"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/localcache"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
	"github.com/onizukazaza/anc-portal-be-fake/server"
)

func main() {
	bootStart := time.Now()

	// >> Graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// >> Logger setup
	appLogger := log.New("anc-portal-api")
	log.Set(appLogger)

	// >> Load Config
	cfg, err := config.Load()
	if err != nil {
		appLogger.Fatal().Err(err).Msg("config load failed")
	}

	// >> Swagger Runtime Override (ค่าจาก env แทนค่าตอน swag init)
	if cfg.Swagger.Enabled {
		env := strings.ToUpper(cfg.StageStatus)
		docs.SwaggerInfo.Title = fmt.Sprintf("ANC Portal API [%s]", env)
		docs.SwaggerInfo.Description = fmt.Sprintf("Backend API for ANC Insurance Portal\n\nEnvironment: %s", env)
		docs.SwaggerInfo.Host = cfg.Swagger.Host
		docs.SwaggerInfo.Schemes = cfg.Swagger.Schemes
		docs.SwaggerInfo.BasePath = cfg.Swagger.BasePath
	}

	// >> Run migration only on local stage with explicit env flag
	// Env required: RUN_DB_MIGRATION=true
	if cfg.StageStatus == "local" && os.Getenv("RUN_DB_MIGRATION") == "true" {
		migrationURL := &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(cfg.Database.User, cfg.Database.Password),
			Host:     fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port),
			Path:     cfg.Database.DBName,
			RawQuery: fmt.Sprintf("sslmode=%s", cfg.Database.SSLMode),
		}
		if migrateErr := postgres.MigrateUp(migrationURL.String(), "migrations"); migrateErr != nil {
			appLogger.Fatal().Err(migrateErr).Msg("database migration failed")
		}
	}

	// >> OpenTelemetry setup (optional — OTLP/HTTP, ไม่ใช้ gRPC)
	otelShutdown, err := appOtel.Init(ctx, cfg.OTel)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("otel init failed")
	}
	defer otelShutdown(ctx)

	// >> Connect database manager (main + external)
	dbManager, err := postgres.NewManager(ctx, cfg)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("database connection failed")
	}
	defer dbManager.Close()

	// >> Kafka setup (optional)
	var producer *kafka.Producer

	if cfg.Kafka.Enabled {
		producer, err = kafka.NewProducer(kafka.ProducerConfig{
			Brokers:      cfg.Kafka.Brokers,
			Topic:        cfg.Kafka.Topic,
			WriteTimeout: cfg.Kafka.WriteTimeout,
		})
		if err != nil {
			appLogger.Fatal().Err(err).Msg("kafka producer init failed")
		}
		defer producer.Close()
	}

	// >> Redis cache setup (optional)
	var cacheClient *cache.Client

	if cfg.Redis.Enabled {
		cacheClient, err = cache.New(ctx, cache.Config{
			Host:        cfg.Redis.Host,
			Port:        cfg.Redis.Port,
			Password:    cfg.Redis.Password,
			DB:          cfg.Redis.DB,
			KeyPrefix:   cfg.Redis.KeyPrefix,
			OtelEnabled: cfg.OTel.Enabled,
		})
		if err != nil {
			appLogger.Fatal().Err(err).Msg("redis connection failed")
		}
		defer cacheClient.Close()
		appLogger.Info().Str("addr", fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)).Msg("redis connected")
	}

	// >> Local cache setup (optional — in-memory, otter)
	var lc *localcache.Client

	if cfg.LocalCache.Enabled {
		lc, err = localcache.New(localcache.Config{
			MaxSize: cfg.LocalCache.MaxSize,
			TTL:     cfg.LocalCache.TTL,
		})
		if err != nil {
			appLogger.Fatal().Err(err).Msg("local cache init failed")
		}
		defer lc.Close()
		appLogger.Info().Int("max_size", cfg.LocalCache.MaxSize).Dur("ttl", cfg.LocalCache.TTL).Msg("local cache initialized")
	}

	// >> Print startup banner
	bannerRows := []banner.Row{
		// >> Runtime & Build
		banner.GoRow(),
		banner.HostRow(),
		banner.BuildRow(buildinfo.GitCommit, buildinfo.BuildTime),

		// >> Main Database
		banner.DBRow("Database (main)", cfg.Database.DBName, cfg.Database.Host, cfg.Database.Port),
		banner.DBPoolRow("  Pool (main)", cfg.Database.MaxConns, cfg.Database.MinConns),
	}

	// >> External Databases (visually separated section)
	if len(cfg.ExternalDBs) > 0 {
		bannerRows = append(bannerRows, banner.SectionRow("External Databases"))
		for name := range cfg.ExternalDBs {
			dbCfg := cfg.ExternalDBs[name]
			bannerRows = append(bannerRows, banner.ExtDBRow(name, dbCfg.DBName, dbCfg.Host, dbCfg.Port))
		}
	}

	// >> Infrastructure
	bannerRows = append(bannerRows,
		banner.KafkaRow(cfg.Kafka.Enabled, cfg.Kafka.Brokers, cfg.Kafka.Topic),
		banner.RedisRow(cfg.Redis.Enabled, cfg.Redis.Host, cfg.Redis.Port),
		banner.OTelRow(cfg.OTel.Enabled, cfg.OTel.ExporterURL),
		banner.LocalCacheRow(cfg.LocalCache.Enabled, cfg.LocalCache.MaxSize, cfg.LocalCache.TTL),
		banner.RateLimitRow(cfg.Server.RateLimit.Enabled, cfg.Server.RateLimit.Max, cfg.Server.RateLimit.Expiration),
		banner.SwaggerRow(cfg.Swagger.Enabled, cfg.Swagger.BasePath),
		banner.ServerRow(cfg.Server.Timeout, cfg.Server.BodyLimit),
	)
	banner.Print(banner.Options{
		AppName:  "ANC Portal API",
		Version:  "1.0.0",
		Env:      cfg.StageStatus,
		Port:     cfg.Server.Port,
		BootTime: time.Since(bootStart),
		Rows:     bannerRows,
	})

	srv := server.New(cfg, dbManager, producer, cacheClient, lc)
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		appLogger.Fatal().Err(err).Msg("server exited with error")
	}
}
