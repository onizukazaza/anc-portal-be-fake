// Package middleware รวม Fiber middleware ที่ใช้ร่วมกันใน server
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// AccessLogConfig ตั้งค่า access log middleware
type AccessLogConfig struct {
	// SkipPaths คือ path ที่ไม่ต้อง log (เช่น /healthz, /ready, /metrics)
	SkipPaths []string
}

// AccessLog สร้าง middleware สำหรับ log ทุก HTTP request/response ด้วย zerolog
//
// Log fields:
//   - method, path, status, latency_ms, ip, request_id, user_agent, bytes_in, bytes_out
//
// Skip paths ที่ไม่ต้อง log (health check, metrics) เพื่อลด log noise
func AccessLog(cfg ...AccessLogConfig) fiber.Handler {
	skipSet := make(map[string]struct{})
	if len(cfg) > 0 {
		for _, p := range cfg[0].SkipPaths {
			skipSet[p] = struct{}{}
		}
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// skip health/metrics paths
		if _, skip := skipSet[path]; skip {
			return c.Next()
		}

		start := time.Now()

		// process request
		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()

		event := log.L().Info()
		if status >= 500 {
			event = log.L().Error()
		} else if status >= 400 {
			event = log.L().Warn()
		}

		event.
			Str("method", c.Method()).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", c.IP()).
			Str("request_id", c.GetRespHeader(fiber.HeaderXRequestID)).
			Str("user_agent", c.Get(fiber.HeaderUserAgent)).
			Int("bytes_in", len(c.Body())).
			Int("bytes_out", len(c.Response().Body())).
			Msg("access")

		return err
	}
}
