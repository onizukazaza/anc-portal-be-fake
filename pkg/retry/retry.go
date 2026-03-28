package retry

import (
	"context"
	"fmt"
	"time"
)

// Do รัน fn ซ้ำตาม opts จนสำเร็จหรือหมด retry
//
//	err := retry.Do(ctx, func(ctx context.Context) error {
//	    return callExternalAPI(ctx)
//	}, retry.MaxAttempts(5), retry.Backoff(2*time.Second))
func Do(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	cfg := defaults()
	for _, o := range opts {
		o(&cfg)
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
		if err := fn(ctx); err != nil {
			lastErr = err

			if ctx.Err() != nil {
				return lastErr
			}

			// attempt สุดท้าย — ไม่ต้อง wait
			if attempt == cfg.maxAttempts {
				break
			}

			wait := cfg.backoffFn(attempt, cfg.baseDelay)
			select {
			case <-ctx.Done():
				return lastErr
			case <-time.After(wait):
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("failed after %d attempts: %w", cfg.maxAttempts, lastErr)
}

// ───────────────────────────────────────────────────────────────────
// Options
// ───────────────────────────────────────────────────────────────────

type config struct {
	maxAttempts int
	baseDelay   time.Duration
	backoffFn   BackoffFunc
}

// BackoffFunc คำนวณ delay สำหรับ attempt ที่ n (1-based)
type BackoffFunc func(attempt int, base time.Duration) time.Duration

// Option ปรับแต่ง retry behavior
type Option func(*config)

func defaults() config {
	return config{
		maxAttempts: 3,
		baseDelay:   1 * time.Second,
		backoffFn:   ExponentialBackoff,
	}
}

// MaxAttempts จำนวน attempt สูงสุด (default: 3)
func MaxAttempts(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.maxAttempts = n
		}
	}
}

// Backoff ระยะเวลา base delay (default: 1s)
func Backoff(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.baseDelay = d
		}
	}
}

// WithBackoffFunc ใช้ backoff function ที่กำหนดเอง
func WithBackoffFunc(fn BackoffFunc) Option {
	return func(c *config) {
		if fn != nil {
			c.backoffFn = fn
		}
	}
}

// ───────────────────────────────────────────────────────────────────
// Built-in Backoff Strategies
// ───────────────────────────────────────────────────────────────────

// ExponentialBackoff — 1s, 2s, 4s, 8s ... (base * 2^(attempt-1))
func ExponentialBackoff(attempt int, base time.Duration) time.Duration {
	return base * time.Duration(1<<(attempt-1))
}

// ConstantBackoff — ใช้ delay เท่าเดิมทุก attempt
func ConstantBackoff(_ int, base time.Duration) time.Duration {
	return base
}

// LinearBackoff — 1s, 2s, 3s, 4s ... (base * attempt)
func LinearBackoff(attempt int, base time.Duration) time.Duration {
	return base * time.Duration(attempt)
}
