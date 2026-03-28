// pkg/otel/middleware.go — Fiber middleware สำหรับ distributed tracing
//
// ทุก request จะถูกสร้าง span อัตโนมัติ พร้อม attributes:
//   - http.method, http.route, http.status_code
//   - request_id จาก X-Request-ID header
//   - error status ถ้า response >= 500
package otel

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// ------------------------------
// >> Middleware Config
// ------------------------------

// MiddlewareConfig ปรับแต่ง tracing middleware
type MiddlewareConfig struct {
	// SkipPaths — paths ที่ไม่ต้อง trace เช่น /healthz, /metrics
	SkipPaths map[string]bool
}

// DefaultMiddlewareConfig คือค่า default ที่ skip health + metrics
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		SkipPaths: map[string]bool{
			"/healthz": true,
			"/ready":   true,
			"/metrics": true,
		},
	}
}

// ------------------------------
// >> Fiber Tracing Middleware
// ------------------------------

// Middleware สร้าง span สำหรับทุก incoming HTTP request
func Middleware(cfgs ...MiddlewareConfig) fiber.Handler {
	cfg := DefaultMiddlewareConfig()
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	tracer := Tracer(TracerFiber)

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// >> Skip paths ที่ไม่ต้องการ trace
		if cfg.SkipPaths[path] {
			return c.Next()
		}

		// >> Extract parent context จาก incoming headers (W3C Trace Context)
		ctx := c.UserContext()
		carrier := &fiberHeaderCarrier{ctx: c}
		ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

		// >> สร้าง span ใหม่
		spanName := fmt.Sprintf("%s %s", c.Method(), path)
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Method()),
				semconv.URLPath(path),
				semconv.ServerAddress(c.Hostname()),
			),
		)
		defer span.End()

		// >> Inject trace context เข้า Fiber context
		c.SetUserContext(ctx)

		// >> Record request_id ถ้ามี
		if reqID := c.Get("X-Request-ID"); reqID != "" {
			span.SetAttributes(attribute.String("request_id", reqID))
		}

		start := time.Now()

		// >> Execute next handler
		err := c.Next()

		// >> Record response attributes
		statusCode := c.Response().StatusCode()
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(statusCode),
			attribute.Int64("http.duration_ms", time.Since(start).Milliseconds()),
		)

		// >> Mark span as error ถ้า status >= 500
		if statusCode >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		return err
	}
}

// ------------------------------
// >> Header Carrier for Fiber
// ------------------------------

// fiberHeaderCarrier bridges Fiber headers กับ OTel propagation
type fiberHeaderCarrier struct {
	ctx *fiber.Ctx
}

func (c *fiberHeaderCarrier) Get(key string) string {
	return c.ctx.Get(key)
}

func (c *fiberHeaderCarrier) Set(key, value string) {
	c.ctx.Set(key, value)
}

func (c *fiberHeaderCarrier) Keys() []string {
	keys := make([]string, 0)
	c.ctx.Request().Header.VisitAll(func(k, _ []byte) {
		keys = append(keys, string(k))
	})
	return keys
}

// compile check
var _ propagation.TextMapCarrier = (*fiberHeaderCarrier)(nil)
