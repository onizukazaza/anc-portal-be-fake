// pkg/otel/tracername.go — Central Tracer Name Registry
//
// รวม tracer name ทั้งโปรเจกต์ไว้ที่เดียว เพื่อ:
//   - ไล่ code ง่าย — grep "TracerXxx" เจอทุก span ของ module นั้น
//   - ป้องกัน typo — compile error แทนที่จะเป็น silent wrong name
//   - เห็นภาพรวม instrumentation ทั้ง system ได้ทันที
//
// Naming Convention:
//
//	Infrastructure: "anc/{component}"    เช่น "anc/fiber", "anc/kafka"
//	Module Layer:   "{module}.{layer}"   เช่น "auth.handler", "cmi.service"
//
// วิธีใช้:
//
//	import appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
//
//	ctx, span := appOtel.Tracer(appOtel.TracerAuthService).Start(ctx, "Login")
//	defer span.End()
package otel

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// ─────────────────────────────────────────────────────────────
// >> Infrastructure Tracers — pkg layer
// ─────────────────────────────────────────────────────────────

const (
	TracerFiber      = "anc/fiber"      // HTTP middleware (Fiber)
	TracerHTTPClient = "anc/httpclient" // Outgoing HTTP calls
	TracerKafka      = "anc/kafka"      // Kafka producer & consumer
)

// ─────────────────────────────────────────────────────────────
// >> Module Tracers — internal/modules layer
// ─────────────────────────────────────────────────────────────

const (
	// >> Auth
	TracerAuthHandler = "auth.handler"
	TracerAuthService = "auth.service"
	TracerAuthRepo    = "auth.repository"

	// >> CMI (Compulsory Motor Insurance)
	TracerCMIHandler = "cmi.handler"
	TracerCMIService = "cmi.service"
	TracerCMIRepo    = "cmi.repository"

	// >> External DB
	TracerExtDBHandler = "externaldb.handler"
	TracerExtDBService = "externaldb.service"

	// >> Quotation
	TracerQuotationHandler = "quotation.handler"
	TracerQuotationService = "quotation.service"
	TracerQuotationRepo    = "quotation.repository"

	// >> Webhook (GitHub → Discord)
	TracerWebhookHandler  = "webhook.handler"
	TracerWebhookService  = "webhook.service"
	TracerWebhookNotifier = "webhook.notifier"
)

// ─────────────────────────────────────────────────────────────
// >> Tracer Cache — หลีกเลี่ยง map lookup ซ้ำทุก request
// ─────────────────────────────────────────────────────────────
//
// otel.Tracer() ภายในทำ map lookup ทุกครั้งที่เรียก
// เราเก็บ cache ไว้ใน sync.Map เพื่อคืน trace.Tracer instance เดิม
// ลด overhead บน hot path (ทุก incoming request / ทุก DB query)

var tracerCache sync.Map // map[string]trace.Tracer

// Tracer คืน trace.Tracer จาก cache หรือสร้างใหม่ครั้งแรกแล้ว cache
// ใช้แทน otel.Tracer() โดยตรง — ได้ performance ดีกว่าบน hot path
func Tracer(name string) trace.Tracer {
	if v, ok := tracerCache.Load(name); ok {
		return v.(trace.Tracer)
	}

	t := otel.Tracer(name)
	tracerCache.Store(name, t)
	return t
}
