package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTracer_ReturnsSameInstance(t *testing.T) {
	// >> Setup: use real TracerProvider ให้ tracer มี identity จริง
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(context.Background())
	otel.SetTracerProvider(tp)

	// >> ล้าง cache ก่อนเทสต์
	tracerCache.Range(func(key, _ any) bool {
		tracerCache.Delete(key)
		return true
	})

	t1 := Tracer(TracerAuthHandler)
	t2 := Tracer(TracerAuthHandler)
	t3 := Tracer(TracerCMIService)

	// >> ชื่อเดียวกันต้องได้ instance เดียวกัน (pointer equal)
	if t1 != t2 {
		t.Error("Tracer cache should return same instance for same name")
	}

	// >> ชื่อต่างกันต้องได้คนละ instance
	if t1 == t3 {
		t.Error("Tracer should return different instances for different names")
	}
}

func TestTracer_AllConstantsAreNonEmpty(t *testing.T) {
	// >> ป้องกัน typo — ทุก constant ต้องไม่เป็น empty string
	constants := map[string]string{
		"TracerFiber":            TracerFiber,
		"TracerHTTPClient":       TracerHTTPClient,
		"TracerKafka":            TracerKafka,
		"TracerAuthHandler":      TracerAuthHandler,
		"TracerAuthService":      TracerAuthService,
		"TracerAuthRepo":         TracerAuthRepo,
		"TracerCMIHandler":       TracerCMIHandler,
		"TracerCMIService":       TracerCMIService,
		"TracerCMIRepo":          TracerCMIRepo,
		"TracerExtDBHandler":     TracerExtDBHandler,
		"TracerExtDBService":     TracerExtDBService,
		"TracerQuotationHandler": TracerQuotationHandler,
		"TracerQuotationService": TracerQuotationService,
		"TracerQuotationRepo":    TracerQuotationRepo,
	}

	for name, value := range constants {
		if value == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestTracer_NoDuplicateNames(t *testing.T) {
	// >> ป้องกัน copy-paste — ทุก constant ต้องไม่ซ้ำกัน
	constants := []string{
		TracerFiber,
		TracerHTTPClient,
		TracerKafka,
		TracerAuthHandler,
		TracerAuthService,
		TracerAuthRepo,
		TracerCMIHandler,
		TracerCMIService,
		TracerCMIRepo,
		TracerExtDBHandler,
		TracerExtDBService,
		TracerQuotationHandler,
		TracerQuotationService,
		TracerQuotationRepo,
	}

	seen := make(map[string]bool, len(constants))
	for _, v := range constants {
		if seen[v] {
			t.Errorf("duplicate tracer name: %q", v)
		}
		seen[v] = true
	}
}
