package kafka

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ───────────────────────────────────────────────────────────────────
// kafkaHeaderCarrier unit tests
// ───────────────────────────────────────────────────────────────────

func TestHeaderCarrier_SetAndGet(t *testing.T) {
	var c kafkaHeaderCarrier
	c.Set("traceparent", "00-abc123-def456-01")

	got := c.Get("traceparent")
	if got != "00-abc123-def456-01" {
		t.Fatalf("Get(traceparent) = %q, want %q", got, "00-abc123-def456-01")
	}
}

func TestHeaderCarrier_GetMissing(t *testing.T) {
	var c kafkaHeaderCarrier
	if got := c.Get("nonexistent"); got != "" {
		t.Fatalf("Get(nonexistent) = %q, want empty", got)
	}
}

func TestHeaderCarrier_SetOverwrite(t *testing.T) {
	var c kafkaHeaderCarrier
	c.Set("traceparent", "old-value")
	c.Set("traceparent", "new-value")

	got := c.Get("traceparent")
	if got != "new-value" {
		t.Fatalf("Get after overwrite = %q, want %q", got, "new-value")
	}
	if len(c) != 1 {
		t.Fatalf("carrier length = %d, want 1 (no duplicates)", len(c))
	}
}

func TestHeaderCarrier_Keys(t *testing.T) {
	var c kafkaHeaderCarrier
	c.Set("traceparent", "v1")
	c.Set("tracestate", "v2")

	keys := c.Keys()
	if len(keys) != 2 {
		t.Fatalf("Keys() length = %d, want 2", len(keys))
	}

	want := map[string]bool{"traceparent": true, "tracestate": true}
	for _, k := range keys {
		if !want[k] {
			t.Fatalf("unexpected key %q", k)
		}
	}
}

// ───────────────────────────────────────────────────────────────────
// Inject / Extract roundtrip test
// ───────────────────────────────────────────────────────────────────

func TestTraceContext_InjectExtractRoundtrip(t *testing.T) {
	// Setup: real TracerProvider + W3C propagator
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())

	// Create a span (producer side)
	ctx, span := tp.Tracer("test").Start(context.Background(), "produce")
	producerTraceID := span.SpanContext().TraceID()
	producerSpanID := span.SpanContext().SpanID()

	// Inject trace context into Kafka headers
	headers := injectTraceHeaders(ctx)
	span.End()

	// Verify headers contain traceparent
	carrier := kafkaHeaderCarrier(headers)
	tp_header := carrier.Get("traceparent")
	if tp_header == "" {
		t.Fatal("traceparent header not injected")
	}

	// Extract trace context (consumer side)
	consumerCtx := extractTraceContext(context.Background(), headers)

	// Create a child span using extracted context
	_, childSpan := tp.Tracer("test").Start(consumerCtx, "consume")
	defer childSpan.End()

	// Verify: child span has the same trace ID as the producer
	childTraceID := childSpan.SpanContext().TraceID()
	if childTraceID != producerTraceID {
		t.Fatalf("trace ID mismatch: producer=%s, consumer=%s", producerTraceID, childTraceID)
	}

	// Verify: child span's parent is the producer span
	childParentSpanID := childSpan.(interface{ Parent() trace.SpanContext }).Parent().SpanID()
	if childParentSpanID != producerSpanID {
		t.Fatalf("parent span ID mismatch: expected=%s, got=%s", producerSpanID, childParentSpanID)
	}
}

func TestTraceContext_ExtractWithoutHeaders(t *testing.T) {
	// When no trace headers exist, extractTraceContext should return a valid context
	// with no span (or a noop span) — should not panic
	ctx := extractTraceContext(context.Background(), nil)
	if ctx == nil {
		t.Fatal("extractTraceContext returned nil context")
	}

	// No active span expected
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		t.Fatal("expected invalid span context when no headers provided")
	}
}
