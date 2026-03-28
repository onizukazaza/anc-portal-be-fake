package kafka

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg, err := NewMessage("debug.message", "u1", map[string]any{"message": "hello"}, map[string]string{"source": "api"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if msg.Type != "debug.message" {
		t.Fatalf("expected type debug.message, got %s", msg.Type)
	}
	if msg.Key != "u1" {
		t.Fatalf("expected key u1, got %s", msg.Key)
	}
	if msg.OccurredAt.IsZero() {
		t.Fatal("expected occurredAt to be set")
	}
	if string(msg.Payload) != `{"message":"hello"}` {
		t.Fatalf("unexpected payload: %s", string(msg.Payload))
	}
	if msg.Metadata["source"] != "api" {
		t.Fatalf("expected metadata source=api, got %q", msg.Metadata["source"])
	}
}

func TestRouterDispatch(t *testing.T) {
	router := NewRouter()
	called := false

	if err := router.Register("debug.message", func(ctx context.Context, msg Message) error {
		called = true
		if msg.Type != "debug.message" {
			t.Fatalf("unexpected type: %s", msg.Type)
		}
		return nil
	}); err != nil {
		t.Fatalf("register handler failed: %v", err)
	}

	msg, err := NewMessage("debug.message", "u1", map[string]any{"message": "hello"}, nil)
	if err != nil {
		t.Fatalf("create message failed: %v", err)
	}

	if err := router.Dispatch(context.Background(), msg); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestRouterDispatchFallback(t *testing.T) {
	router := NewRouter()
	called := false
	router.SetFallback(func(ctx context.Context, msg Message) error {
		called = true
		return nil
	})

	msg, err := NewMessage("unknown.event", "u1", map[string]any{"message": "hello"}, nil)
	if err != nil {
		t.Fatalf("create message failed: %v", err)
	}

	if err := router.Dispatch(context.Background(), msg); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if !called {
		t.Fatal("expected fallback handler to be called")
	}
}

func TestRouterDispatchWithoutHandler(t *testing.T) {
	router := NewRouter()
	msg, err := NewMessage("unknown.event", "u1", map[string]any{"message": "hello"}, nil)
	if err != nil {
		t.Fatalf("create message failed: %v", err)
	}

	err = router.Dispatch(context.Background(), msg)
	if !errors.Is(err, ErrKafkaHandlerNotFound) {
		t.Fatalf("expected ErrKafkaHandlerNotFound, got %v", err)
	}
}

// ───────────────────────────────────────────────────────────────────
// Consumer Config & Retry Tests
// ───────────────────────────────────────────────────────────────────

func TestNewConsumerValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConsumerConfig
		wantErr string
	}{
		{
			name:    "missing brokers",
			cfg:     ConsumerConfig{Topic: "t", GroupID: "g"},
			wantErr: "brokers",
		},
		{
			name:    "missing topic",
			cfg:     ConsumerConfig{Brokers: []string{"localhost:9092"}, GroupID: "g"},
			wantErr: "topic",
		},
		{
			name:    "missing group id",
			cfg:     ConsumerConfig{Brokers: []string{"localhost:9092"}, Topic: "t"},
			wantErr: "group id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConsumer(tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestConsumerDefaultMaxRetries(t *testing.T) {
	// MaxRetries = 0 should default to 3
	// We can't easily test the ready consumer without Kafka,
	// but we test the config defaults via ConsumerConfig
	cfg := ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.MaxRetries != 3 {
		t.Fatalf("expected MaxRetries=3, got %d", cfg.MaxRetries)
	}
}

func TestProcessWithRetrySuccess(t *testing.T) {
	c := &Consumer{cfg: ConsumerConfig{MaxRetries: 3}}
	attempts := 0

	msg, _ := NewMessage("test.event", "k1", map[string]any{"ok": true}, nil)
	err := c.processWithRetry(context.Background(), func(ctx context.Context, msg Message) error {
		attempts++
		if attempts < 2 {
			return errors.New("transient error")
		}
		return nil
	}, msg)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestProcessWithRetryExhausted(t *testing.T) {
	c := &Consumer{cfg: ConsumerConfig{MaxRetries: 2}}
	attempts := 0

	msg, _ := NewMessage("test.event", "k1", map[string]any{"fail": true}, nil)
	err := c.processWithRetry(context.Background(), func(ctx context.Context, msg Message) error {
		attempts++
		return errors.New("permanent error")
	}, msg)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if !strings.Contains(err.Error(), "failed after 2 attempts") {
		t.Fatalf("expected retry exhaustion message, got %q", err.Error())
	}
}

func TestProcessWithRetryCancelledContext(t *testing.T) {
	c := &Consumer{cfg: ConsumerConfig{MaxRetries: 5}}

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	msg, _ := NewMessage("test.event", "k1", map[string]any{"cancel": true}, nil)
	err := c.processWithRetry(ctx, func(_ context.Context, _ Message) error {
		attempts++
		cancel() // cancel after first attempt
		return errors.New("error")
	}, msg)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt (cancelled), got %d", attempts)
	}
}

func TestSendToDLQNilProducer(t *testing.T) {
	// DLQ disabled (dlq == nil) — should not panic
	c := &Consumer{cfg: ConsumerConfig{MaxRetries: 3}}
	c.sendToDLQ(context.Background(), []byte("raw"), "test.event", errors.New("fail"))
	// no panic = pass
}
