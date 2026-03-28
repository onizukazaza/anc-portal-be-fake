// run cmd 🔑
// go run ./cmd/worker/main.go
//
// cmd/worker — Kafka consumer process ที่ทำงานแยกจาก API server
//
// หน้าที่หลัก:
//   - รับ message จาก Kafka topic แล้ว dispatch ไปยัง handler ที่ลงทะเบียนไว้ผ่าน Router
//   - แต่ละ event type (เช่น "debug.message") จะมี handler เฉพาะของตัวเองที่สามารถเพิ่มได้ผ่าน router.Register()
//   - message ที่ไม่มี handler จะถูก log เตือนผ่าน fallback handler แทนที่จะ error
//
// ต้องการ:
//   - Kafka broker ที่เชื่อมต่อได้ (ตั้งค่าผ่าน KAFKA_BROKERS, KAFKA_TOPIC, KAFKA_GROUP_ID)
//   - KAFKA_ENABLED=true ใน env ถ้าเป็น false worker จะ exit ทันที
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/banner"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/kafka"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
	"github.com/rs/zerolog"
)

func main() {
	bootStart := time.Now()

	// >> Graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// >> Logger setup
	workerLogger := log.New("anc-portal-worker")
	log.Set(workerLogger)

	// >> Load Config
	cfg, err := config.Load()
	if err != nil {
		workerLogger.Fatal().Err(err).Msg("config load failed")
	}

	// >> Guard: exit if Kafka is disabled
	if !cfg.Kafka.Enabled {
		workerLogger.Info().Msg("kafka disabled; worker exiting")
		return
	}

	// >> OpenTelemetry setup (optional — เชื่อม trace ข้าม Producer → Consumer)
	otelShutdown, err := appOtel.Init(ctx, cfg.OTel)
	if err != nil {
		workerLogger.Fatal().Err(err).Msg("otel init failed")
	}
	defer otelShutdown(ctx)

	// >> Init Kafka consumer (with DLQ support)
	consumer, err := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers:    cfg.Kafka.Brokers,
		Topic:      cfg.Kafka.Topic,
		GroupID:    cfg.Kafka.GroupID,
		DLQTopic:   cfg.Kafka.DLQTopic,
		MaxRetries: cfg.Kafka.MaxRetries,
		MaxBytes:   cfg.Kafka.MaxBytes,
	})
	if err != nil {
		workerLogger.Fatal().Err(err).Msg("kafka consumer init failed")
	}
	defer consumer.Close()

	// >> Setup event router + fallback
	router := kafka.NewRouter()
	router.SetFallback(func(ctx context.Context, msg kafka.Message) error {
		log.L().Warn().Str("event_type", msg.Type).Str("key", msg.Key).Msg("kafka message skipped: no handler registered")
		return nil
	})

	// >> Register event handlers
	if err := router.Register("debug.message", func(ctx context.Context, msg kafka.Message) error {
		var payload map[string]any
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return err
		}

		log.L().Info().Str("event_type", msg.Type).Str("key", msg.Key).Any("payload", payload).Msg("kafka message received")
		return nil
	}); err != nil {
		workerLogger.Fatal().Err(err).Msg("register kafka handler failed")
	}

	// >> Start consuming messages (retry + DLQ จัดการใน consumer)
	banner.Print(banner.Options{
		AppName:  "ANC Portal Worker",
		Version:  "1.0.0",
		Env:      cfg.StageStatus,
		BootTime: time.Since(bootStart),
		Rows: []banner.Row{
			banner.GoRow(),
			banner.HostRow(),
			banner.BuildRow(buildinfo.GitCommit, buildinfo.BuildTime),
			banner.KafkaRow(true, cfg.Kafka.Brokers, cfg.Kafka.Topic),
			{Label: "Group ID", Value: cfg.Kafka.GroupID},
			{Label: "DLQ Topic", Value: cfg.Kafka.DLQTopic},
			{Label: "Max Retries", Value: fmt.Sprintf("%d", cfg.Kafka.MaxRetries)},
			banner.OTelRow(cfg.OTel.Enabled, cfg.OTel.ExporterURL),
			{Label: "Health Port", Value: "20001"},
		},
	})

	// >> Health probe HTTP server สำหรับ K8s liveness/readiness
	healthSrv := startHealthProbe(consumer, workerLogger)
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = healthSrv.Shutdown(shutCtx)
	}()

	if err := consumer.StartMessages(ctx, func(ctx context.Context, msg kafka.Message) error {
		return router.Dispatch(ctx, msg)
	}); err != nil {
		workerLogger.Fatal().Err(err).Msg("worker stopped with error")
	}
}

// startHealthProbe เปิด HTTP health probe server บน port 20001
// สำหรับ K8s liveness/readiness — ตรวจว่า consumer ยังทำงาน
func startHealthProbe(consumer *kafka.Consumer, logger *zerolog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if consumer.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not_ready"}`))
		}
	})

	srv := &http.Server{
		Addr:              ":20001",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("health probe server failed")
		}
	}()

	return srv
}
