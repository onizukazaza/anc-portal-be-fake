package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/auth"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/webhook"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/enum"
	module "github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/validator"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/cache"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/kafka"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/localcache"
	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
	mw "github.com/onizukazaza/anc-portal-be-fake/server/middleware"
)

type KafkaProducer interface {
	PublishMessage(ctx context.Context, msg kafka.Message) error
}

type Server struct {
	cfg           *config.Config
	db            database.Provider
	cache         cache.Cache
	localCache    localcache.Cache
	hybridCache   *localcache.Hybrid // L1 (otter) + L2 (Redis) — nil ถ้าไม่มีทั้งคู่
	app           *fiber.App
	kafkaProducer KafkaProducer
	onShutdown    []func() // callbacks to run before server fully stops
}

const (
	defaultShutdownTimeout = 10 * time.Second
	healthTimeout          = 2 * time.Second
)

// ------------------------------
// Constructor
// ------------------------------
func New(cfg *config.Config, db database.Provider, producer KafkaProducer, cacheClient cache.Cache, lc localcache.Cache) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
		BodyLimit:    cfg.Server.BodyLimit,
		ErrorHandler: globalErrorHandler,
	})

	s := &Server{cfg: cfg, db: db, cache: cacheClient, localCache: lc, app: app, kafkaProducer: producer}

	// สร้าง hybrid cache เมื่อมีทั้ง local + redis
	if lc != nil && cacheClient != nil {
		s.hybridCache = localcache.NewHybrid(lc, cacheClient)
	}

	s.initMiddlewares()
	s.registerRoutes()

	return s
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	// >> Graceful shutdown hook
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		_ = s.app.ShutdownWithContext(shutdownCtx)

		// Wait for in-flight background work (e.g. webhook notifications)
		for _, fn := range s.onShutdown {
			fn()
		}
	}()

	// >> Listen and serve
	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	go func() {
		errCh <- s.app.Listen(addr)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		if err != nil && err.Error() == "server is not running" {
			return nil
		}
		return err
	}
}

// checkDependencies verifies the database and cache (if configured) are reachable.
// Used by both /healthz and /ready endpoints.
func (s *Server) checkDependencies(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, healthTimeout)
	defer cancel()

	if err := s.db.HealthCheck(ctx); err != nil {
		return err
	}
	if s.cache != nil {
		if err := s.cache.Ping(ctx); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
	}
	return nil
}

func (s *Server) initMiddlewares() {
	// >> Recover from panics
	s.app.Use(recover.New())

	// >> Request tracing and response compression
	s.app.Use(requestid.New())

	// >> Access logging (skip noisy endpoints)
	s.app.Use(mw.AccessLog(mw.AccessLogConfig{
		SkipPaths: []string{"/healthz", "/ready", "/metrics"},
	}))

	s.app.Use(compress.New())

	// >> OpenTelemetry tracing middleware (skip health/metrics)
	if s.cfg.OTel.Enabled {
		s.app.Use(appOtel.Middleware())
	}

	// >> CORS setup from config
	allowOrigins := "*"
	if len(s.cfg.Server.AllowOrigins) > 0 {
		allowOrigins = strings.Join(s.cfg.Server.AllowOrigins, ",")
	}

	s.app.Use(cors.New(cors.Config{
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type, Authorization, X-Request-ID",
		AllowOrigins: allowOrigins,
	}))

	// >> Rate limiting (config-driven)
	if s.cfg.Server.RateLimit.Enabled {
		max := s.cfg.Server.RateLimit.Max
		if max <= 0 {
			max = 100
		}
		expiration := s.cfg.Server.RateLimit.Expiration
		if expiration <= 0 {
			expiration = time.Minute
		}
		s.app.Use(limiter.New(limiter.Config{
			Max:        max,
			Expiration: expiration,
			LimitReached: func(c *fiber.Ctx) error {
				return dto.Error(c, fiber.StatusTooManyRequests, "rate limit exceeded")
			},
			SkipFailedRequests: false,
			Next: func(c *fiber.Ctx) bool {
				path := c.Path()
				return path == "/healthz" || path == "/ready" || path == "/metrics"
			},
		}))
	}
}

func (s *Server) registerRoutes() {
	// >> Prometheus metrics endpoint (Grafana scrape target)
	if s.cfg.OTel.Enabled {
		s.app.Get("/metrics", adaptor.HTTPHandler(appOtel.PrometheusHandler()))
	}

	// >> Swagger routes (config-driven, docs generated by swag init)
	if s.cfg.Swagger.Enabled {
		s.app.Get("/swagger/*", fiberSwagger.WrapHandler)
	}

	// >> Health endpoint (includes database + redis health check)
	s.app.Get("/healthz", func(c *fiber.Ctx) error {
		if err := s.checkDependencies(c.UserContext()); err != nil {
			return dto.Error(c, fiber.StatusServiceUnavailable, "degraded: "+err.Error())
		}
		return dto.Success(c, fiber.StatusOK, fiber.Map{"status": enum.HealthOK})
	})
	s.app.Get("/ready", func(c *fiber.Ctx) error {
		if err := s.checkDependencies(c.UserContext()); err != nil {
			return dto.Error(c, fiber.StatusServiceUnavailable, enum.HealthNotReady+": "+err.Error())
		}
		return dto.Success(c, fiber.StatusOK, fiber.Map{
			"status":    enum.HealthReady,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// >> Module routes
	api := s.app.Group("/v1")

	// ─── Auth strategies ─────────────────────────────────────────
	tokenSigner := auth.NewTokenSigner(s.cfg)
	jwtAuth := mw.Auth(mw.AuthConfig{TokenSigner: tokenSigner})
	apiKeyAuth := mw.APIKey(mw.APIKeyConfig{ValidKeys: s.cfg.Server.APIKeys.Internal})

	deps := module.Deps{
		Config:      s.cfg,
		DB:          s.db,
		Cache:       s.cache,
		LocalCache:  s.localCache,
		HybridCache: s.hybridCache,
		Middleware: module.Middleware{
			JWTAuth:    jwtAuth,
			APIKeyAuth: apiKeyAuth,
		},
	}

	// ─── Module registration (each module applies auth internally) ─
	auth.Register(api, deps, tokenSigner)
	if wait := webhook.Register(api, deps); wait != nil {
		s.onShutdown = append(s.onShutdown, wait)
	}
	externaldb.Register(api, deps)
	quotation.Register(api, deps)
	cmi.Register(api, deps)

	// >> Kafka test route for local stage (public, no auth)
	if s.cfg.StageStatus == enum.StageLocal && s.kafkaProducer != nil {
		api.Post("/kafka/publish", s.publishKafka)
	}
}

func (s *Server) publishKafka(c *fiber.Ctx) error {
	type req struct {
		Key       string            `json:"key"`
		EventType string            `json:"eventType"`
		Message   string            `json:"message"`
		Metadata  map[string]string `json:"metadata"`
	}

	var body req
	if err := c.BodyParser(&body); err != nil {
		return dto.Error(c, fiber.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(body.Message) == "" {
		return dto.Error(c, fiber.StatusBadRequest, "message is required")
	}

	eventType := strings.TrimSpace(body.EventType)
	if eventType == "" {
		eventType = "debug.message"
	}

	msg, err := kafka.NewMessage(eventType, body.Key, map[string]any{
		"message": body.Message,
	}, body.Metadata)
	if err != nil {
		return dto.Error(c, fiber.StatusInternalServerError, "failed to build message payload")
	}

	if err := s.kafkaProducer.PublishMessage(c.UserContext(), msg); err != nil {
		return dto.Error(c, fiber.StatusBadGateway, err.Error())
	}

	return dto.SuccessWithMessage(c, fiber.StatusAccepted, "published", nil)
}

// globalErrorHandler is the Fiber-level error handler for all unhandled errors.
//   - ErrValidation: response was already written by BindAndValidate — do nothing.
//   - *fiber.Error:  use the status code from the error.
//   - other errors:  respond with 500 Internal Server Error.
func globalErrorHandler(c *fiber.Ctx, err error) error {
	if errors.Is(err, validator.ErrValidation) {
		return nil // response already written
	}

	code := fiber.StatusInternalServerError
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code = fe.Code
	}

	return c.Status(code).JSON(dto.ApiResponse{
		Status:     enum.ResponseError,
		StatusCode: code,
		Message:    err.Error(),
	})
}
