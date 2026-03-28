// pkg/kafka/tracing.go — W3C Trace Context propagation สำหรับ Kafka messages
//
// ใช้ Kafka message headers เป็น carrier สำหรับ inject/extract trace context
// ทำให้ trace เชื่อมต่อกันได้: Producer span → Kafka → Consumer span
//
// เมื่อ OTel ไม่ได้เปิด จะใช้ noop propagator — ไม่มี overhead
package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

// ─────────────────────────────────────────────────────────────
// Header Carrier — bridges Kafka headers กับ OTel propagation
// ─────────────────────────────────────────────────────────────

// kafkaHeaderCarrier adapts []kafka.Header to the propagation.TextMapCarrier
// interface for injecting/extracting W3C trace context.
type kafkaHeaderCarrier []kafka.Header

func (c *kafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaHeaderCarrier) Set(key, value string) {
	// Overwrite if key already exists
	for i, h := range *c {
		if h.Key == key {
			(*c)[i].Value = []byte(value)
			return
		}
	}
	*c = append(*c, kafka.Header{Key: key, Value: []byte(value)})
}

func (c *kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c))
	for i, h := range *c {
		keys[i] = h.Key
	}
	return keys
}

// ─────────────────────────────────────────────────────────────
// Inject / Extract helpers
// ─────────────────────────────────────────────────────────────

// injectTraceHeaders สร้าง Kafka headers ที่มี trace context ของ span ปัจจุบัน
func injectTraceHeaders(ctx context.Context) []kafka.Header {
	var carrier kafkaHeaderCarrier
	otel.GetTextMapPropagator().Inject(ctx, &carrier)
	return carrier
}

// extractTraceContext อ่าน trace context จาก Kafka message headers
// แล้วคืน context ที่มี parent span จาก producer
func extractTraceContext(ctx context.Context, headers []kafka.Header) context.Context {
	carrier := kafkaHeaderCarrier(headers)
	return otel.GetTextMapPropagator().Extract(ctx, &carrier)
}
