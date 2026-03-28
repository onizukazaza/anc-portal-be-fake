# Software Architecture — ANC Portal Backend

> **v2.0** — Last updated: March 2026
>
> Style: **Modular Monolith + Hexagonal Architecture (Ports & Adapters)**
>
> Deploy ง่ายแบบ Monolith แต่โครงสร้างภายในแยก module ชัดเจน
> พร้อม extract เป็น Microservice ได้ทุกเมื่อ

---

## สารบัญ

1. [ภาพรวมระบบ](#1-ภาพรวมระบบ)
2. [Hexagonal Architecture](#2-hexagonal-architecture)
3. [Tech Stack](#3-tech-stack)
4. [Data Flow ตัวอย่าง](#4-data-flow-ตัวอย่าง)
5. [Database — Multi-DB Provider](#5-database--multi-db-provider)
6. [Cache — Hybrid L1 + L2](#6-cache--hybrid-l1--l2)
7. [Event-Driven — Kafka](#7-event-driven--kafka)
8. [Observability — OpenTelemetry](#8-observability--opentelemetry)
9. [Data Sync Framework](#9-data-sync-framework)
10. [Reusable Packages](#10-reusable-packages)
11. [Shared Infrastructure](#11-shared-infrastructure)
12. [Design Patterns](#12-design-patterns)
13. [Scalability Path](#13-scalability-path)
14. [จุดแข็งและจุดอ่อน](#14-จุดแข็งและจุดอ่อน)

---

## 1. ภาพรวมระบบ

```
┌───────────────────────────────────────────────────────────────────────────┐
│                          Go Binary (single)                               │
│                                                                           │
│  ┌────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ ┌────────┐ ┌────────┐   │
│  │  Auth  │ │Quotation │ │ExternalDB│ │  CMI   │ │Payment │ │  Job   │   │
│  │ module │ │  module  │ │  module  │ │ module │ │(future)│ │(future)│   │
│  └───┬────┘ └────┬─────┘ └────┬─────┘ └───┬────┘ └────────┘ └────────┘   │
│      │           │             │            │                              │
│  ┌───┴───────────┴─────────────┴────────────┴────────────────────────────┐│
│  │                     Shared Infrastructure                             ││
│  │  pagination · dto · enum · utils · module.Deps                        ││
│  └───┬──────────┬──────────┬────────────┬──────────┬─────────────────────┘│
│      │          │          │            │          │                       │
│  ┌───┴───┐ ┌───┴───┐ ┌───┴────┐ ┌─────┴─────┐ ┌──┴──────┐               │
│  │  pgx  │ │ Redis │ │ Kafka  │ │   OTel    │ │  HTTP   │               │
│  │ Pool  │ │+Otter │ │+Tracing│ │  Tracing  │ │ Client  │               │
│  └───────┘ └───────┘ └────────┘ └───────────┘ └─────────┘               │
└───────────────────────────────────────────────────────────────────────────┘
     ▼            ▼            ▼             ▼
  PostgreSQL    Redis       Kafka      Tempo/Prometheus/Grafana
```

### 6 Entry Points

binary เดียว แยก command ตามหน้าที่:

| Command | หน้าที่ | วิธีรัน |
|---|---|---|
| `cmd/api` | HTTP server (Fiber) — REST API หลัก | `.\run.ps1 dev` |
| `cmd/worker` | Kafka consumer — งาน background | `.\run.ps1 worker` |
| `cmd/migrate` | Database migration (golang-migrate) | `.\run.ps1 migrate` |
| `cmd/seed` | Seed ข้อมูลเริ่มต้น | `.\run.ps1 seed` |
| `cmd/import` | CSV import (insurer, user, province) | `go run ./cmd/import --help` |
| `cmd/sync` | Data sync จาก External DB → Main DB | `go run ./cmd/sync --help` |

### Startup Banner

ทุก binary แสดง startup banner พร้อม runtime info:

```
 ┌──────────────────────────────────────────────────────────┐
 │                ANC Portal API v1.0.0                     │
 │                       [LOCAL]                            │
 ├──────────────────────────────────────────────────────────┤
 │  Port ················ :20000                             │
 │  Go ·················· go1.25.0 windows/amd64            │
 │  Host / PID ·········· DESKTOP-ABC / 12345               │
 │  Build ··············· a1b2c3d (2026-03-28T10:30:00Z)    │
 │  Database (main) ····· ✔ anc_portal @ localhost:5432     │
 │    Pool (main) ······· ✔ 20 max / 5 min conns           │
 │─── External Databases ───────────────────────────────────│
 │  DB ↗ partner ········ ⬡ partner_db @ 10.0.0.5:5432     │
 │  Kafka ··············· ✔ localhost:9092 → anc-topic      │
 │  Redis ··············· ✔ localhost:6379                   │
 │  Rate Limit ·········· ✔ 100 req / 1m                    │
 │  Swagger ············· ✔ /v1/swagger                      │
 ├──────────────────────────────────────────────────────────┤
 │                  Boot time: 125ms                        │
 └──────────────────────────────────────────────────────────┘
```

Build info (`Build` row) ถูก inject ตอน `go build` ผ่าน `-ldflags`:
- **Local dev**: แสดง `dev (no build info)`
- **CI/CD build**: แสดง git commit hash + timestamp

---

## 2. Hexagonal Architecture

ทุก module มีโครงสร้าง **เหมือนกัน** ตาม hexagonal pattern:

```
module/
├── module.go          ← จุดลงทะเบียน (wire dependencies + routes)
├── domain/            ← Business entities + rules (ไม่พึ่ง framework)
├── ports/             ← Interfaces (สัญญาที่ domain ต้องการ)
├── app/               ← Use cases / service (orchestrate logic)
└── adapters/          ← Implementation จริง
    ├── http/          ← Controller + Handler (Fiber)
    └── postgres/      ← Repository (pgx)
```

### Dependency Rule — ไหลเข้าด้านในเท่านั้น

```
adapters → app → ports ← domain
   ▲                        ▲
   │   adapters implement   │
   └────── ports ───────────┘
```

| Layer | กฎ |
|---|---|
| `domain/` | ไม่ import อะไรเลย — pure Go structs & logic |
| `ports/` | define interfaces — ไม่ import adapters |
| `app/` | ใช้ ports interfaces — ไม่รู้จัก postgres/http |
| `adapters/` | implement ports — รู้จัก framework (Fiber, pgx) |

### ตัวอย่าง — Quotation Module

```
quotation/
├── module.go                           // Register(deps) → wire repo, service, routes
├── domain/quotation.go                 // Quotation struct, Status enum
├── ports/repository.go                 // interface QuotationRepository
├── app/service.go                      // QuotationService.ListByCustomer()
└── adapters/
    ├── http/controller.go + handler.go // GET /quotations/:id, GET /quotations?...
    └── postgres/repository.go          // SQL queries with pgx
```

### วิธีเพิ่ม Module ใหม่

1. สร้างโฟลเดอร์ `internal/modules/{name}/`
2. สร้างโครงสร้าง: `domain/`, `ports/`, `app/`, `adapters/http/`, `adapters/postgres/`
3. สร้าง `module.go` ที่ implement `Module` interface
4. ลงทะเบียนใน `server/server.go`
5. เพิ่ม tracer constant ใน `pkg/otel/tracername.go` (ถ้าต้องการ tracing)

---

## 3. Tech Stack

| Layer | เทคโนโลยี | เหตุผล |
|---|---|---|
| **Language** | Go 1.25 | Performance, concurrency, single binary |
| **HTTP** | Fiber v2.52 | เร็ว, middleware chain, Go-friendly |
| **Database** | PostgreSQL + pgx v5 | Connection pool, prepared stmt cache, multi-DB |
| **Cache L1** | Otter (in-memory) | ~1μs lookup, lock-free, S3-FIFO eviction |
| **Cache L2** | Redis (go-redis v9) | ~1ms lookup, shared ข้าม instance |
| **Messaging** | Kafka (segmentio) | Event-driven, KRaft mode, DLQ support |
| **HTTP Client** | pkg/httpclient | Connection pool, retry, OTel tracing, circuit breaker |
| **Retry** | pkg/retry | Exponential/Constant/Linear backoff |
| **Logging** | zerolog | Structured, env-aware (console/JSON) |
| **Observability** | OpenTelemetry SDK | Traces → Tempo, Metrics → Prometheus |
| **Config** | Viper | 12-factor, env vars, feature toggles |
| **Deploy** | Docker + Kubernetes | Kustomize base + overlays (staging/prod) |
| **API Docs** | Swagger/OpenAPI | Auto-generated จาก code annotations |

---

## 4. Data Flow ตัวอย่าง

### List Quotations

```
Client → GET /v1/quotations?customer_id=C001&page=1&limit=20
         │
         ▼
    [Fiber Middleware] recover → requestid → access_log → compress → otel → cors → limiter
         │
         ▼
    [Handler] ParsePagination(c) → validate query params
         │
         ▼
    [Service] ListByCustomer(ctx, customerID, pagination)
         │
         ▼
    [Repository] pagination.From("quotations")
                     .Select("id","doc_no","total_amount","status","created_at")
                     .Where("customer_id = $1")
                     .Search("doc_no", "customer_name")
                     .Paginate(pg, "created_at", allowedSorts)
                     .DataSQL()
         │
         ▼
    [Response] { data: [...], pagination: { page:1, limit:20, total:150 } }
```

### Kafka Cross-Service Trace

```
API Server (Producer)                     Worker (Consumer)
[Handler] → [Service] → [Kafka Publish]  →  [Kafka Process] → [Business Handler]
    └──── Fiber span        └── inject        └── extract ──── same Trace ID ─┘
                              W3C headers       W3C headers
```

---

## 5. Database — Multi-DB Provider

```go
type Provider interface {
    Main() *pgxpool.Pool              // Primary DB (anc-portal)
    External(name string) *pgxpool.Pool // Legacy DBs (meprakun ฯลฯ)
    Read() *pgxpool.Pool              // Read replica (future)
    Write() *pgxpool.Pool             // Write master (future)
}
```

| Config | ค่า | หน้าที่ |
|---|---|---|
| Max Connections | 20 | จำกัดจำนวน connection สูงสุด |
| Min Connections | 5 | รักษา warm connections |
| Statement Timeout | 5s | ป้องกัน query ค้าง |
| Prepared Statements | อัตโนมัติ | pgx จัดการ — DB ไม่ parse SQL ซ้ำ |

External databases แสดงแยกชัดใน banner ด้วยไอคอน `↗` และสี magenta

---

## 6. Cache — Hybrid L1 + L2

```
Request → L1 Otter (~1μs) → L2 Redis (~1ms) → Database (~5ms)
              ▲                    ▲
              │   backfill on miss │
              └────────────────────┘
```

| กลยุทธ์ | พฤติกรรม |
|---|---|
| **Read-through** | L1 miss → ดึง L2 → backfill L1 |
| **Write-through** | เขียนทั้ง L1 + L2 พร้อมกัน |
| **Feature toggles** | `REDIS_ENABLED`, `LOCAL_CACHE_ENABLED` — ปิดได้อิสระ |

ดูรายละเอียดเพิ่มเติม: [Redis Cache Guide](../integrations/redis-cache-guide.md)

---

## 7. Event-Driven — Kafka

```
Producer (API)                 Kafka Topic               Consumer (Worker)
┌─────────────┐                                      ┌──────────────────┐
│ fire & forget│ ────────▶  [message + W3C headers]  ─▶│ EventRouter      │
│ + OTel span  │                                      │  .Handle(type,fn)│
└─────────────┘                                      └────────┬─────────┘
                                                              │
                                                     ┌────────▼─────────┐
                                                     │ success → commit  │
                                                     │ fail    → retry   │
                                                     │ max retry → DLQ   │
                                                     └──────────────────┘
```

| Feature | รายละเอียด |
|---|---|
| **KRaft mode** | ไม่ต้องพึ่ง ZooKeeper |
| **Event Router** | dispatch by event type |
| **Message Envelope** | `Message{Type, Key, Payload, Metadata, OccurredAt}` |
| **Dead Letter Queue** | failed messages หลัง retry ครบ → DLQ topic |
| **W3C Trace Propagation** | inject/extract `traceparent` header ข้าม services |
| **Optional** | `KAFKA_ENABLED=false` ปิดได้ |

---

## 8. Observability — OpenTelemetry

### Central Tracer Name Registry

ทุก tracer name รวมไว้ที่ `pkg/otel/tracername.go` — ป้องกัน typo, compile-time check:

```go
// Infrastructure tracers
TracerFiber      = "anc/fiber"
TracerHTTPClient = "anc/httpclient"
TracerKafka      = "anc/kafka"

// Module tracers: "{module}.{layer}"
TracerAuthHandler = "auth.handler"
TracerCMIService  = "cmi.service"
// ... etc.
```

Cached ใน `sync.Map` — lock-free read, zero allocation หลัง warm-up

### Naming Convention

| ประเภท | Pattern | ตัวอย่าง |
|---|---|---|
| **Tracer name** | `{module}.{layer}` | `auth.handler`, `cmi.service` |
| **Span name** | `{Operation}` เท่านั้น | `Login`, `GetPolicyByJobID` |
| **Infrastructure** | `anc/{component}` | `anc/fiber`, `anc/kafka` |

ดูรายละเอียดเพิ่มเติม: [OTel Tracing Guide](../integrations/otel-tracing-guide.md)

---

## 9. Data Sync Framework

```
cmd/sync (CLI)
     │
     ▼
  Runner.RunOne("quotations", req)
     │
     ▼
  Registry.Get("quotations") → QuotationSyncer
     │
     ▼
  External DB (meprakun) ──batch 500──→ Main DB (UPSERT)
```

| Mode | พฤติกรรม |
|---|---|
| **Full** | ลบข้อมูลเดิมทั้งหมด + insert ใหม่ |
| **Incremental** | sync เฉพาะ rows ที่ `updated_at > Since` |

```bash
# Full sync
go run ./cmd/sync --env .env.local --table quotations --mode full

# Incremental (last 24h)
go run ./cmd/sync --env .env.local --table quotations --mode incremental --since 24h
```

เพิ่ม Syncer ใหม่: implement `Syncer` interface → register ใน `registerSyncers()`

---

## 10. Reusable Packages

### pkg/httpclient — HTTP Client

```go
client := httpclient.New(
    httpclient.BaseURL("https://api.example.com"),
    httpclient.Timeout(10*time.Second),
    httpclient.WithRetry(3),
    httpclient.WithTracing(),          // OTel tracing อัตโนมัติ
    httpclient.WithCircuitBreaker("my-api"),  // Circuit Breaker (sony/gobreaker)
)
client.GetJSON(ctx, "/path", &result)
client.PostJSON(ctx, "/path", body, &result)

// ตรวจสอบสถานะ circuit
if httpclient.IsCircuitOpen(err) {
    // fallback logic
}
```

**Execution chain:** traced → circuit breaker → retry (5xx only)

### pkg/retry — Retry Strategies

| Strategy | พฤติกรรม |
|---|---|
| `ExponentialBackoff` | 1s → 2s → 4s → 8s (default) |
| `ConstantBackoff` | delay คงที่ทุกครั้ง |
| `LinearBackoff` | 1s → 2s → 3s → 4s |

### pkg/log — Structured Logging

- **Local/UAT**: colored console output
- **Production**: JSON format
- ตรวจจับจาก `STAGE_STATUS` env อัตโนมัติ

### pkg/banner — Startup Banner

- ANSI box-drawing + color
- Component status rows (✔ enabled / ✗ disabled)
- Section separator สำหรับ external databases
- Build info จาก `-ldflags` (Git commit + build time)
- Zero dependency

### pkg/buildinfo — Build Information

```go
// ถูก inject ตอน compile ผ่าน -ldflags:
var (
    GitCommit = "dev"
    BuildTime = "unknown"
)
```

---

## 11. Shared Infrastructure

### Module Dependencies

```go
type Deps struct {
    Config      *config.Config
    DB          database.Provider
    Cache       cache.Cache         // L2 Redis
    LocalCache  localcache.Cache    // L1 Otter
    HybridCache *localcache.Hybrid  // L1+L2
}

type Module interface {
    Register(router fiber.Router, deps Deps)
}
```

### Standard Response

```go
ApiResponse{
    Status:     "OK",
    StatusCode: 200,
    Message:    "success",
    Result:     ResultData{ Data: ..., Meta: ... },
}
```

Helpers: `Success()`, `SuccessWithMessage()`, `SuccessWithMeta()`, `Error()`

### Pagination — Fluent SQL Builder

```go
pg := pagination.FromFiber(c) // ?page=1&limit=20&sort=created_at&order=desc

query := pagination.From("quotations").
    Select("id", "doc_no", "total_amount").
    Where("customer_id = $1").
    Search("doc_no", "customer_name").
    Paginate(pg, "created_at", allowedSorts)
```

Response: `Response[T]{Data, Page, Limit, Total, TotalPages, HasNext, HasPrev}`

### Utilities (Go Generics)

| Package | Functions |
|---|---|
| `utils/id.go` | `NewID(prefix)` → `"usr-20260326-a1b2c3d4"` |
| `utils/pointer.go` | `Ptr[T]()`, `Deref[T]()`, `DerefOr[T]()` |
| `utils/slice.go` | `Contains[T]()`, `Unique[T]()`, `Map[A,B]()`, `Filter[T]()` |
| `utils/json.go` | `PrettyJSON()`, `MaskJSON()`, `SafeUnmarshal()` |
| `utils/string.go` | `TrimLower()`, `Truncate()`, `DefaultIfEmpty()` |

---

## 12. Design Patterns

| Pattern | ใช้ที่ไหน |
|---|---|
| **Hexagonal / Ports & Adapters** | ทุก module — domain ไม่พึ่ง framework |
| **Repository** | `ports/repository.go` → `adapters/postgres/` |
| **Dependency Injection** | `module.Deps` → ส่ง config, DB, cache เข้า module |
| **Strategy** | `TokenSigner` interface — swap impl ได้ |
| **Builder (Fluent)** | `pagination.From().Select().Where().Paginate()` |
| **Functional Options** | `httpclient.New(BaseURL(), Timeout(), WithRetry())` |
| **Provider** | `database.Provider` — multi-DB access |
| **Feature Toggle** | Redis, Kafka, OTel — on/off ผ่าน env |
| **Read-Through Cache** | Hybrid L1 → L2 → DB |
| **Event Router** | Kafka consumer dispatch by event type |
| **Dead Letter Queue** | Kafka DLQ — failed messages หลัง retry |
| **Registry** | `sync.Registry` — pluggable syncer registration |
| **Module Registration** | `module.go` → `Register(deps)` per module |
| **Generics** | `Ptr[T]`, `Map[A,B]`, `Response[T]` — Go 1.18+ |
| **Tracer Registry** | `pkg/otel/tracername.go` — central constants + sync.Map cache |
| **Table-Driven Tests + testkit** | unit test ทุกตัว — generic assertions, hand-written fakes, zero external deps |

---

## 13. Scalability Path

```
Phase 1 (ปัจจุบัน)          Phase 2                Phase 3              Phase 4
────────────────────  ──────────────────    ─────────────────    ──────────────────
  Single binary        Horizontal scale      Extract modules      Full microservices
  1 main DB +          Multiple API pods     Quotation → gRPC     Event sourcing
  N external DBs       Read replica DB       Module per binary    Service mesh
  Optional infra       Redis cluster         Shared → library     Independent DBs
  Rate limiting        CDN + WAF
  OTel tracing         Log aggregation
```

---

## 14. จุดแข็งและจุดอ่อน

### จุดแข็ง

- Module แยกชัด — เพิ่ม module ใหม่ง่าย ไม่กระทบของเดิม
- Deploy ง่าย — binary เดียว, Docker image เดียว
- Infrastructure optional — ปิด Redis/Kafka/OTel ได้ ไม่พัง
- Test ง่าย — mock ports interface ได้เลย
- Scale path ชัด — จาก monolith ไป microservice ได้เป็นขั้นตอน
- Data Sync framework — pluggable syncer, Full/Incremental mode
- Reusable packages — httpclient, retry, cache ใช้ซ้ำข้าม project ได้
- Distributed Tracing — OTel ครบทุก layer รวม Kafka cross-service
- Central Tracer Registry — compile-time check, cached, เปิดไฟล์เดียวเห็นทั้งระบบ
- Kafka DLQ — failed messages ไม่หาย จัดการได้ภายหลัง
- Rate Limiting — ป้องกัน abuse ด้วย Fiber limiter middleware
- Startup Banner — เห็น runtime info ทั้งหมดตอน boot ไม่ต้องค้น log
- Circuit Breaker — sony/gobreaker ใน httpclient ป้องกัน cascade failure
- Request Validation — go-playground/validator ผ่าน BindAndValidate helper
- Access Logging — zerolog structured logging ทุก request พร้อม latency, status, request_id
- Worker Health Probe — HTTP /healthz สำหรับ K8s liveness/readiness
- Test Infrastructure — `internal/testkit` generic assertions + hand-written fakes ไม่พึ่ง external deps ([ดูคู่มือ](../testing/unit-test-guide.md))

### จุดอ่อน

- Cache ยังไม่ได้ใช้ใน service layer (มี interface พร้อมแล้ว)
- JWT ยังเป็น simple signer (ไม่มี RSA/ECDSA)
- Unit test coverage ยังไม่ครบทุก layer

---

> **v2.0** — March 2026 | ANC Portal Backend Team
