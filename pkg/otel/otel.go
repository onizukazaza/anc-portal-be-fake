// pkg/otel — OpenTelemetry bootstrap สำหรับ Traces + Metrics
//
// ออกแบบให้ยืดหยุ่น:
//   - ใช้ OTLP/HTTP exporter (ไม่ใช้ gRPC) — เบา, ไม่ต้องพึ่ง protobuf stub
//   - Prometheus endpoint สำหรับ Grafana scrape metrics โดยตรง
//   - เปิด/ปิดผ่าน OTEL_ENABLED ถ้าปิด ระบบยังทำงานปกติ (noop)
//   - Sampler ปรับได้ผ่าน OTEL_SAMPLE_RATIO (0.0 – 1.0)
//
// วิธีใช้:
//
//	shutdown, err := otel.Init(ctx, cfg.OTel)
//	defer shutdown(ctx)
package otel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	promClient "github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// ShutdownFunc ใช้ defer เพื่อ flush ข้อมูลก่อนปิด app
type ShutdownFunc func(ctx context.Context)

// ------------------------------
// >> Init — Bootstrap OpenTelemetry
// ------------------------------

// Init สร้าง TracerProvider + MeterProvider แล้ว register เป็น global
// คืน ShutdownFunc สำหรับ graceful shutdown
func Init(ctx context.Context, cfg config.OTel) (ShutdownFunc, error) {
	if !cfg.Enabled {
		log.L().Info().Msg("otel disabled; using noop providers")
		return func(_ context.Context) {}, nil
	}

	// >> Build resource (service identity)
	res, err := buildResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	// >> Init trace exporter (OTLP/HTTP)
	traceShutdown, err := initTracer(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("otel tracer: %w", err)
	}

	// >> Init metric exporter (Prometheus + OTLP/HTTP)
	metricShutdown, err := initMeter(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("otel meter: %w", err)
	}

	// >> Set global propagator (W3C TraceContext + Baggage)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.L().Info().
		Str("service", cfg.ServiceName).
		Str("exporter", cfg.ExporterURL).
		Float64("sample_ratio", cfg.SampleRatio).
		Msg("otel initialized (OTLP/HTTP)")

	// >> Combined shutdown
	shutdown := func(ctx context.Context) {
		log.L().Info().Msg("otel shutting down...")
		traceShutdown(ctx)
		metricShutdown(ctx)
	}

	return shutdown, nil
}

// ------------------------------
// >> Resource Builder
// ------------------------------

func buildResource(cfg config.OTel) (*resource.Resource, error) {
	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	}

	if cfg.Env != "" {
		attrs = append(attrs, resource.WithAttributes(
			attribute.String("deployment.environment.name", cfg.Env),
		))
	}
	if cfg.Release != "" {
		attrs = append(attrs, resource.WithAttributes(
			semconv.ServiceVersion(cfg.Release),
		))
	}

	return resource.New(context.Background(), attrs...)
}

// ------------------------------
// >> Trace Provider (OTLP/HTTP)
// ------------------------------

func initTracer(ctx context.Context, cfg config.OTel, res *resource.Resource) (ShutdownFunc, error) {
	// >> สร้าง OTLP/HTTP exporter — ไม่ใช้ gRPC
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.ExporterURL),
		otlptracehttp.WithInsecure(), // local dev ไม่ต้อง TLS
	)
	if err != nil {
		return nil, err
	}

	// >> Sampler — ควบคุมอัตราการ sample traces
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.SampleRatio),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// >> Register as global TracerProvider
	otel.SetTracerProvider(tp)

	return func(ctx context.Context) {
		if err := tp.Shutdown(ctx); err != nil {
			log.L().Error().Err(err).Msg("otel trace provider shutdown failed")
		}
	}, nil
}

// ------------------------------
// >> Metric Provider (Prometheus + OTLP/HTTP)
// ------------------------------

func initMeter(ctx context.Context, cfg config.OTel, res *resource.Resource) (ShutdownFunc, error) {
	// >> Prometheus exporter — expose /metrics endpoint สำหรับ Grafana scrape
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("prometheus exporter: %w", err)
	}

	// >> OTLP/HTTP metric exporter — ส่ง metrics ไป OTel Collector ด้วย
	otlpExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(cfg.ExporterURL),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(otlpExporter,
				sdkmetric.WithInterval(15*time.Second),
			),
		),
	)

	// >> Register as global MeterProvider
	otel.SetMeterProvider(mp)

	return func(ctx context.Context) {
		if err := mp.Shutdown(ctx); err != nil {
			log.L().Error().Err(err).Msg("otel meter provider shutdown failed")
		}
	}, nil
}

// ------------------------------
// >> Prometheus HTTP Handler
// ------------------------------

// PrometheusHandler คืน http.Handler สำหรับ expose /metrics endpoint
// ใช้กับ Fiber ผ่าน adaptor หรือ stand-alone HTTP server
func PrometheusHandler() http.Handler {
	return promClient.Handler()
}
