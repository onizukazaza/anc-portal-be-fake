package httpclient

import (
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/retry"
	"github.com/sony/gobreaker/v2"
)

// ──────────────────────────────────────────────────────────────────
// config — internal configuration struct
// ──────────────────────────────────────────────────────────────────

type config struct {
	baseURL string
	timeout time.Duration
	headers map[string]string

	// connection pool
	connectTimeout      time.Duration
	maxIdleConns        int
	maxIdleConnsPerHost int
	idleConnTimeout     time.Duration

	// retry
	retryEnabled bool
	retryOpts    []retry.Option

	// tracing
	tracingEnabled bool

	// circuit breaker
	cbEnabled  bool
	cbSettings gobreaker.Settings
}

func defaultConfig() config {
	return config{
		timeout:             30 * time.Second,
		connectTimeout:      5 * time.Second,
		maxIdleConns:        100,
		maxIdleConnsPerHost: 10,
		idleConnTimeout:     90 * time.Second,
		headers:             make(map[string]string),
		tracingEnabled:      true, // เปิดเป็น default — ปิดได้ด้วย WithoutTracing()
	}
}

// ───────────────────────────────────────────────────────────────────
// Option — functional options pattern
// ───────────────────────────────────────────────────────────────────

// Option ใช้ config Client ก่อนสร้าง
type Option func(*config)

// BaseURL กำหนด base URL สำหรับทุก request (e.g. "https://api.example.com")
func BaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// Timeout กำหนด total timeout ของทั้ง request (default: 30s)
func Timeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// ConnectTimeout กำหนด timeout สำหรับ TCP connection (default: 5s)
func ConnectTimeout(d time.Duration) Option {
	return func(c *config) { c.connectTimeout = d }
}

// WithHeader เพิ่ม default header ที่จะใส่ทุก request
func WithHeader(key, value string) Option {
	return func(c *config) { c.headers[key] = value }
}

// WithHeaders เพิ่ม default headers หลายตัวพร้อมกัน
func WithHeaders(headers map[string]string) Option {
	return func(c *config) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// ───────────────────────────────────────────────────────────────────
// Retry Options
// ───────────────────────────────────────────────────────────────────

// WithRetry เปิด retry พร้อมกำหนดจำนวนครั้งสูงสุด (ใช้ exponential backoff default)
func WithRetry(maxAttempts int) Option {
	return func(c *config) {
		c.retryEnabled = true
		c.retryOpts = []retry.Option{
			retry.MaxAttempts(maxAttempts),
			retry.Backoff(500 * time.Millisecond),
			retry.WithBackoffFunc(retry.ExponentialBackoff),
		}
	}
}

// WithRetryOptions เปิด retry พร้อม options ที่กำหนดเอง — ยืดหยุ่นเต็มที่
func WithRetryOptions(opts ...retry.Option) Option {
	return func(c *config) {
		c.retryEnabled = true
		c.retryOpts = opts
	}
}

// ───────────────────────────────────────────────────────────────────
// Connection Pool Options
// ───────────────────────────────────────────────────────────────────

// MaxIdleConns กำหนดจำนวน idle connections สูงสุดของ pool (default: 100)
func MaxIdleConns(n int) Option {
	return func(c *config) { c.maxIdleConns = n }
}

// MaxIdleConnsPerHost กำหนดจำนวน idle connections ต่อ host (default: 10)
func MaxIdleConnsPerHost(n int) Option {
	return func(c *config) { c.maxIdleConnsPerHost = n }
}

// IdleConnTimeout กำหนดเวลาที่ idle connection จะถูกปิด (default: 90s)
func IdleConnTimeout(d time.Duration) Option {
	return func(c *config) { c.idleConnTimeout = d }
}

// ───────────────────────────────────────────────────────────────────
// Tracing Options
// ───────────────────────────────────────────────────────────────────

// WithoutTracing ปิด OTel tracing (default เปิดอยู่)
func WithoutTracing() Option {
	return func(c *config) { c.tracingEnabled = false }
}

// ───────────────────────────────────────────────────────────────────
// Circuit Breaker Options
// ───────────────────────────────────────────────────────────────────

// WithCircuitBreaker เปิด circuit breaker ด้วย default settings
// (name = "httpclient", MaxRequests = 5, Interval = 30s, Timeout = 10s)
//
// Circuit breaker จะ "เปิด" (open) เมื่อ request ล้มเหลวต่อเนื่อง
// และจะ "ปิด" (closed) อีกครั้งหลังจาก Timeout ผ่านไปแล้ว request สำเร็จ
func WithCircuitBreaker(name string) Option {
	return func(c *config) {
		c.cbEnabled = true
		c.cbSettings = gobreaker.Settings{
			Name:        name,
			MaxRequests: 5,
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5
			},
		}
	}
}

// WithCircuitBreakerSettings เปิด circuit breaker ด้วย settings ที่กำหนดเอง
func WithCircuitBreakerSettings(settings gobreaker.Settings) Option {
	return func(c *config) {
		c.cbEnabled = true
		c.cbSettings = settings
	}
}
