# OpenTelemetry (OTel) — Distributed Tracing & Metrics

> **v2.0** — Last updated: March 2026
>
> โครงสร้าง OTel ของโปรเจกต์ `anc-portal-be`
> สำหรับนักพัฒนาที่ต้องเพิ่ม span, ดู trace, หรือแก้ไข observability config
>
> Quick Start: [OTel + Grafana Quick Start](otel-grafana-observability.md)

---

## สารบัญ

1. [ภาพรวมสถาปัตยกรรม](#1-ภาพรวมสถาปัตยกรรม)
2. [Trace Flow — เส้นทางของ Span](#2-trace-flow--เส้นทางของ-span)
3. [โครงสร้างไฟล์](#3-โครงสร้างไฟล์)
4. [Central Tracer Name Registry](#4-central-tracer-name-registry)
5. [วิธีเพิ่ม Span ใหม่](#5-วิธีเพิ่ม-span-ใหม่)
6. [Naming Convention](#6-naming-convention)
7. [Kafka Trace Propagation](#7-kafka-trace-propagation)
8. [Metrics](#8-metrics)
9. [Configuration](#9-configuration)
10. [Observability Stack (Local Dev)](#10-observability-stack-local-dev)
11. [Performance Design](#11-performance-design)
12. [Checklist สำหรับ Code Review](#12-checklist-สำหรับ-code-review)

---

## 1. ภาพรวมสถาปัตยกรรม

```
                          ┌───────────────────────┐
                          │   Grafana (Dashboard)  │
                          │   :3001                │
                          └────┬──────────┬────────┘
                               │          │
                        Traces │          │ Metrics
                               ▼          ▼
                          ┌─────────┐ ┌────────────┐
                          │  Tempo  │ │ Prometheus  │
                          │  :3200  │ │   :9090     │
                          └────┬────┘ └─────┬──────┘
                               │            │
                          OTLP │    remote   │ scrape
                               │    write    │ /metrics
                          ┌────┴────────────┴──────┐
                          │   OTel Collector        │
                          │   :4318 (OTLP/HTTP)     │
                          └────────────┬───────────┘
                                       │
                               OTLP/HTTP export
                                       │
          ┌────────────────────────────┴────────────────────────────┐
          │                                                         │
   ┌──────┴──────┐                                          ┌───────┴──────┐
   │  API Server │                                          │    Worker    │
   │  :20000     │  ──── Kafka (trace headers) ────▶        │  (consumer) │
   └─────────────┘                                          └──────────────┘
```

**เทคโนโลยีที่ใช้:**

| Component | Library / Tool |
|---|---|
| Trace SDK | `go.opentelemetry.io/otel/sdk v1.42.0` |
| Exporter | OTLP/HTTP (ไม่ใช้ gRPC — เบากว่า, ไม่ต้อง protobuf stub) |
| Propagation | W3C TraceContext + Baggage |
| Metrics | Prometheus exporter (local) + OTLP/HTTP (remote) |
| DB tracing | `otelpgx` (auto-trace ทุก SQL query) |
| Redis tracing | `redisotel` (auto-trace ทุก command) |
| Kafka tracing | Custom header carrier (W3C propagation) |
| Trace storage | Grafana Tempo |
| Dashboards | Grafana |

---

## 2. Trace Flow — เส้นทางของ Span

### API Request Flow

```
Client HTTP Request
  │  (traceparent header)
  ▼
┌─────────────────────────────────────────────────┐
│ Fiber Middleware (TracerFiber)                   │
│ Span: "GET /v1/cmi/{job_id}/request-policy..."  │
│ Kind: Server                                    │
└──────────────────┬──────────────────────────────┘
                   │ ctx
                   ▼
        ┌──────────────────────┐
        │ Handler              │
        │ Span: "GetPolicy..." │
        │ TracerCMIHandler     │
        └──────────┬───────────┘
                   │ ctx
                   ▼
        ┌──────────────────────┐
        │ Service              │
        │ Span: "GetPolicy..." │
        │ TracerCMIService     │
        └──────────┬───────────┘
                   │ ctx
              ┌────┴─────────────────┐
              ▼                      ▼
   ┌─────────────────┐   ┌──────────────────────┐
   │ Repository       │   │ HTTP Client (ext)    │
   │ Span: "FindBy.." │   │ Span: "HTTP GET"     │
   │ TracerCMIRepo    │   │ TracerHTTPClient     │
   └────────┬─────────┘   └──────────────────────┘
            │ ctx
            ▼
   ┌─────────────────┐
   │ otelpgx (auto)  │
   │ Span: SQL query  │
   └──────────────────┘
```

### Kafka Cross-Service Flow

```
API Server (Producer)                          Worker (Consumer)
┌──────────────────────────┐                  ┌──────────────────────────┐
│ Span: "orders publish"   │                  │ Span: "orders process"   │
│ Kind: Producer           │  ── Kafka ──▶    │ Kind: Consumer           │
│ TracerKafka              │  (traceparent    │ TracerKafka              │
└──────────────────────────┘   in headers)    └────────────┬─────────────┘
                                                           │ ctx (same trace!)
                                                           ▼
                                               ┌──────────────────────┐
                                               │ Handler (business)    │
                                               │ Span: custom logic    │
                                               └──────────────────────┘
```

> **สำคัญ:** Producer inject W3C `traceparent` header เข้า Kafka message,
> Consumer extract ออกมาเป็น parent context → trace เชื่อมข้าม service โดยอัตโนมัติ

---

## 3. โครงสร้างไฟล์

```
pkg/otel/
├── otel.go              # Init() — bootstrap TracerProvider + MeterProvider
├── middleware.go         # Fiber HTTP middleware (auto-span ทุก request)
├── tracername.go         # ★ Central Tracer Name Registry (constants + cache)
└── tracername_test.go    # Tests สำหรับ registry

pkg/httpclient/
└── client.go            # HTTP client — TracerHTTPClient, W3C inject

pkg/kafka/
├── tracing.go           # Header carrier + inject/extract helpers
├── tracing_test.go      # Roundtrip propagation tests
├── producer.go          # TracerKafka — span per publish
└── consumer.go          # TracerKafka — span per message process

internal/modules/{module}/
├── adapters/http/
│   └── handler.go       # Tracer{Module}Handler
├── app/
│   └── service.go       # Tracer{Module}Service
└── adapters/postgres/
    └── repository.go    # Tracer{Module}Repo
```

---

## 4. Central Tracer Name Registry

> **ไฟล์:** `pkg/otel/tracername.go`

ทุก tracer name ถูกรวมไว้ที่เดียว — ป้องกัน typo, ค้นหาง่าย, compile-time check

### Infrastructure Tracers

```go
const (
    TracerFiber      = "anc/fiber"       // Fiber HTTP middleware
    TracerHTTPClient = "anc/httpclient"  // Outgoing HTTP calls
    TracerKafka      = "anc/kafka"       // Kafka producer & consumer
)
```

### Module Tracers

```go
const (
    // Auth
    TracerAuthHandler = "auth.handler"
    TracerAuthService = "auth.service"
    TracerAuthRepo    = "auth.repository"

    // CMI
    TracerCMIHandler  = "cmi.handler"
    TracerCMIService  = "cmi.service"
    TracerCMIRepo     = "cmi.repository"

    // External DB
    TracerExtDBHandler = "externaldb.handler"
    TracerExtDBService = "externaldb.service"

    // Quotation
    TracerQuotationHandler = "quotation.handler"
    TracerQuotationService = "quotation.service"
    TracerQuotationRepo    = "quotation.repository"
)
```

### วิธีใช้

```go
import appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

// สร้าง span — ใช้ constant จาก registry
ctx, span := appOtel.Tracer(appOtel.TracerAuthService).Start(ctx, "Login")
defer span.End()
```

### ทำไมไม่ใช้ `otel.Tracer()` ตรง ๆ ?

| | `otel.Tracer("string")` | `appOtel.Tracer(constant)` |
|---|---|---|
| Typo | Runtime silent — ชื่อผิดก็ไม่รู้ | **Compile error** |
| ค้นหา span ของ module | `grep -r '"cmi"'` — เจอทุกอย่าง | `grep TracerCMI` — เฉพาะ OTel |
| Performance | Map lookup ทุกครั้ง | **Cached** ใน `sync.Map` |
| ภาพรวม | กระจายทั่ว codebase | **เปิดไฟล์เดียวเห็นทั้ง system** |

---

## 5. วิธีเพิ่ม Span ใหม่

### ขั้นตอนที่ 1: เพิ่ม constant ใน `tracername.go`

```go
// ใน pkg/otel/tracername.go
const (
    // >> Payment (ใหม่)
    TracerPaymentHandler = "payment.handler"
    TracerPaymentService = "payment.service"
    TracerPaymentRepo    = "payment.repository"
)
```

### ขั้นตอนที่ 2: อัปเดต test ใน `tracername_test.go`

เพิ่ม constant ใหม่เข้า `TestTracer_AllConstantsAreNonEmpty` และ `TestTracer_NoDuplicateNames`

### ขั้นตอนที่ 3: ใช้ใน code

```go
package http

import (
    appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
)

func (h *Handler) CreatePayment(c *fiber.Ctx) error {
    // >> สร้าง span
    ctx, span := appOtel.Tracer(appOtel.TracerPaymentHandler).Start(c.UserContext(), "CreatePayment")
    defer span.End()

    // >> เพิ่ม attributes (optional)
    span.SetAttributes(attribute.String("payment.id", paymentID))

    // >> business logic...
    result, err := h.service.Create(ctx, req)
    if err != nil {
        // >> บันทึก error ใน span
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return dto.Error(c, fiber.StatusInternalServerError, "payment failed")
    }

    return dto.Success(c, fiber.StatusCreated, result)
}
```

---

## 6. Naming Convention

### Tracer Name (ระบุ module + layer)

```
Pattern:  "{module}.{layer}"
ตัวอย่าง: "auth.handler", "cmi.service", "quotation.repository"
```

### Span Name (ระบุ operation อย่างเดียว)

```
Pattern:  "{OperationName}"
ตัวอย่าง: "Login", "GetPolicyByJobID", "FindByID"
```

> **ห้าม** ใส่ layer prefix ใน span name เช่น ~~"handler.Login"~~ ~~"repo.FindByID"~~
> เพราะ layer อยู่ใน tracer name อยู่แล้ว

### Infrastructure Span Names

```
Fiber:       "{METHOD} {PATH}"    เช่น "GET /v1/cmi/123/..."
HTTP Client: "HTTP {METHOD}"      เช่น "HTTP POST"
Kafka:       "{topic} publish"    เช่น "orders publish"
             "{topic} process"    เช่น "orders process"
```

---

## 7. Kafka Trace Propagation

### วิธีทำงาน

```
Producer                          Kafka                         Consumer
┌─────────────┐                                          ┌──────────────┐
│ 1. สร้าง     │     ┌─────────────────────────┐         │ 4. Extract   │
│    span      │     │ Message                 │         │    headers    │
│              │     │ ┌─────────────────────┐ │         │              │
│ 2. Inject    │────▶│ │ traceparent: 00-abc │ │────▶    │ 5. สร้าง     │
│    headers   │     │ │ tracestate: ...     │ │         │    child span│
│              │     │ └─────────────────────┘ │         │    (same     │
│ 3. Publish   │     │ payload: {...}          │         │     trace)   │
└─────────────┘     └─────────────────────────┘         └──────────────┘
```

### ไฟล์ที่เกี่ยวข้อง

| ไฟล์ | หน้าที่ |
|---|---|
| `pkg/kafka/tracing.go` | `kafkaHeaderCarrier` — adapter ระหว่าง Kafka headers กับ OTel |
| `pkg/kafka/tracing.go` | `injectTraceHeaders()` — Producer inject W3C headers |
| `pkg/kafka/tracing.go` | `extractTraceContext()` — Consumer extract parent context |
| `pkg/kafka/producer.go` | สร้าง `Producer` span → inject → WriteMessages |
| `pkg/kafka/consumer.go` | Extract → สร้าง `Consumer` span → dispatch handler |

### Result ใน Grafana

Trace จะแสดงเป็น chain เดียว:

```
[API Handler] → [Kafka Publish] → [Kafka Process] → [Business Handler]
     └──────────── same Trace ID ─────────────────────────┘
```

---

## 8. Metrics

### Dual Export Strategy

| ช่องทาง | ใช้ทำอะไร |
|---|---|
| **Prometheus** (`/metrics`) | Grafana scrape โดยตรง — ดู dashboard |
| **OTLP/HTTP** (remote) | ส่งไป OTel Collector → Prometheus remote write |

### Built-in Metrics

OTel SDK สร้าง metrics ให้อัตโนมัติ:

- `http.server.request.duration` — latency ของทุก HTTP request
- `http.server.active_requests` — concurrent requests
- Runtime metrics (goroutine count, GC, memory)

### เข้าดู

```
# Prometheus raw metrics
http://localhost:20000/metrics

# Grafana dashboard
http://localhost:3001
```

---

## 9. Configuration

### Environment Variables

| Variable | Default | คำอธิบาย |
|---|---|---|
| `OTEL_ENABLED` | `false` | เปิด/ปิด OTel ทั้งระบบ |
| `OTEL_SERVICE_NAME` | — | ชื่อ service ใน trace (`service.name`) |
| `OTEL_EXPORTER_URL` | — | OTel Collector endpoint เช่น `localhost:4318` |
| `OTEL_SAMPLE_RATIO` | `1.0` | อัตราการ sample (0.0 – 1.0) |
| `OTEL_RELEASE` | — | Version ของ service (`service.version`) |
| `OTEL_ENV` | — | Environment (`deployment.environment.name`) |

### Sampling แนะนำ

| Environment | `OTEL_SAMPLE_RATIO` | เหตุผล |
|---|---|---|
| Local | `1.0` | ดู trace ทุก request |
| Staging | `0.5` | ดูครึ่งหนึ่ง ลด overhead |
| Production | `0.1` | 10% — ประหยัด storage + bandwidth |

### เมื่อ `OTEL_ENABLED=false`

- `Init()` คืน noop shutdown function
- `otel.Tracer()` คืน noop tracer → span ไม่ถูกสร้าง
- `otel.GetTextMapPropagator()` คืน noop propagator → headers ไม่ถูก inject/extract
- **Zero performance overhead** — ไม่มีค่าใช้จ่ายเลย

---

## 10. Observability Stack (Local Dev)

### เริ่ม Stack

```bash
cd deployments/observability
docker compose up -d
```

### Services

| Service | URL | หน้าที่ |
|---|---|---|
| **Grafana** | http://localhost:3001 | Dashboard + Trace Explorer |
| **Tempo** | http://localhost:3200 | Trace storage (query by trace ID) |
| **Prometheus** | http://localhost:9090 | Metrics storage + alerting |
| **OTel Collector** | http://localhost:4318 | รับ OTLP/HTTP → route to Tempo + Prometheus |

### ดู Traces ใน Grafana

1. เปิด Grafana → http://localhost:3001
2. ไป **Explore** → เลือก **Tempo** datasource
3. Search by:
   - **Service Name**: `anc-portal-dev` (ตาม `OTEL_SERVICE_NAME`)
   - **Span Name**: เช่น `GET /v1/cmi/...`
   - **Trace ID**: copy จาก log หรือ response header

---

## 11. Performance Design

### Tracer Cache (`sync.Map`)

```go
// ❌ ก่อน: map lookup ทุกครั้ง (มี mutex lock ภายใน SDK)
otel.Tracer("auth.handler")

// ✅ หลัง: atomic read จาก sync.Map (lock-free สำหรับ read-heavy)
appOtel.Tracer(appOtel.TracerAuthHandler)
```

`sync.Map` เหมาะกับ pattern นี้เพราะ:
- **Read-heavy**: อ่าน tracer ทุก request, เขียนแค่ครั้งแรก
- **Lock-free reads**: ไม่มี contention บน hot path
- **Zero allocation** หลังจาก warm-up

### OTel SDK Optimizations

| Feature | Config | ผล |
|---|---|---|
| **Batch Exporter** | `WithBatchTimeout(5s)` | ส่ง spans เป็น batch ทุก 5 วินาที ไม่ใช่ทีละอัน |
| **OTLP/HTTP** | (ไม่ใช้ gRPC) | Header-based protocol เบากว่า, ไม่ต้อง protobuf |
| **ParentBased Sampler** | `SampleRatio = 0.1` | เมื่อ parent ไม่ sample → children ก็ไม่ sample |
| **Metric Periodic Reader** | `WithInterval(15s)` | ส่ง metrics ทุก 15 วินาที |

### เมื่อ OTel ปิด (Noop)

ทุกอย่างกลายเป็น noop — ไม่มี overhead:
- `span.End()` = no-op
- `span.SetAttributes()` = no-op
- `otel.GetTextMapPropagator().Inject()` = no-op
- **ไม่ต้อง if/else ตรวจสอบในโค้ด**

---

## 12. Checklist สำหรับ Code Review

เมื่อ review code ที่เกี่ยวกับ OTel ให้ตรวจสอบ:

- [ ] ใช้ constant จาก `tracername.go` (ไม่ใช่ inline string)
- [ ] Span name เป็น operation อย่างเดียว (ไม่มี layer prefix)
- [ ] `defer span.End()` อยู่ถัดจาก `Start()` ทันที
- [ ] Context propagation: ใช้ `ctx` ที่ได้จาก `Start()` ส่งต่อ
- [ ] Error handling: `span.RecordError(err)` + `span.SetStatus(codes.Error, ...)`
- [ ] ถ้าเพิ่ม module ใหม่: อัปเดต `tracername.go` + `tracername_test.go`
- [ ] ถ้าเพิ่ม infrastructure component: ใช้ prefix `"anc/..."` ใน tracer name

---

> **v2.0** — March 2026 | ANC Portal Backend Team
