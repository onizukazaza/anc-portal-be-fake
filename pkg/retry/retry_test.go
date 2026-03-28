package retry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDoSuccess(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoRetryThenSuccess(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	}, MaxAttempts(5), Backoff(10*time.Millisecond))

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestDoExhausted(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), func(ctx context.Context) error {
		attempts++
		return errors.New("permanent")
	}, MaxAttempts(3), Backoff(10*time.Millisecond))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if !strings.Contains(err.Error(), "failed after 3 attempts") {
		t.Fatalf("expected exhaustion message, got %q", err.Error())
	}
}

func TestDoCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	err := Do(ctx, func(_ context.Context) error {
		attempts++
		cancel()
		return errors.New("error")
	}, MaxAttempts(10), Backoff(10*time.Millisecond))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoDefaultOptions(t *testing.T) {
	cfg := defaults()
	if cfg.maxAttempts != 3 {
		t.Fatalf("expected default maxAttempts=3, got %d", cfg.maxAttempts)
	}
	if cfg.baseDelay != 1*time.Second {
		t.Fatalf("expected default baseDelay=1s, got %v", cfg.baseDelay)
	}
}

func TestMaxAttemptsIgnoresInvalid(t *testing.T) {
	cfg := defaults()
	MaxAttempts(0)(&cfg)
	if cfg.maxAttempts != 3 {
		t.Fatalf("expected maxAttempts unchanged at 3, got %d", cfg.maxAttempts)
	}
	MaxAttempts(-1)(&cfg)
	if cfg.maxAttempts != 3 {
		t.Fatalf("expected maxAttempts unchanged at 3, got %d", cfg.maxAttempts)
	}
}

func TestExponentialBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
	}
	for _, tt := range tests {
		got := ExponentialBackoff(tt.attempt, base)
		if got != tt.want {
			t.Errorf("ExponentialBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestConstantBackoff(t *testing.T) {
	base := 500 * time.Millisecond
	for i := 1; i <= 5; i++ {
		got := ConstantBackoff(i, base)
		if got != base {
			t.Errorf("ConstantBackoff(%d) = %v, want %v", i, got, base)
		}
	}
}

func TestLinearBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 300 * time.Millisecond},
	}
	for _, tt := range tests {
		got := LinearBackoff(tt.attempt, base)
		if got != tt.want {
			t.Errorf("LinearBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestWithBackoffFunc(t *testing.T) {
	custom := func(attempt int, base time.Duration) time.Duration {
		return 42 * time.Millisecond
	}

	attempts := 0
	err := Do(context.Background(), func(_ context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("fail")
		}
		return nil
	}, MaxAttempts(3), Backoff(10*time.Millisecond), WithBackoffFunc(custom))

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestDoWrapsOriginalError(t *testing.T) {
	sentinel := errors.New("my-error")
	err := Do(context.Background(), func(_ context.Context) error {
		return sentinel
	}, MaxAttempts(1), Backoff(10*time.Millisecond))

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got %v", err)
	}
}
